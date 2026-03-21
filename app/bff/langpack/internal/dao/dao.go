package dao

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/teamgram/proto/mtproto"

	"github.com/zeromicro/go-zero/core/logx"
)

// langPackVersion is bumped when the server restarts or cache is refreshed.
// Not a real incremental version - clients will re-fetch on version mismatch.
var langPackVersion = int32(3)

// LangPackEntry stores a parsed language pack.
type LangPackEntry struct {
	Strings  []*mtproto.LangPackString
	Version  int32
	LoadedAt time.Time
}

// Dao is the data access object for langpack.
type Dao struct {
	mu       sync.RWMutex
	cache    map[string]*LangPackEntry // key: "platform/langCode" e.g. "ios/en"
	cacheDir string
	client   *http.Client
}

func New(_ interface{}) *Dao {
	// Store cached .strings files under the working directory
	cacheDir := "data/langpack"
	os.MkdirAll(cacheDir, 0755)

	return &Dao{
		cache:    make(map[string]*LangPackEntry),
		cacheDir: cacheDir,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetLanguages returns the hardcoded list of supported languages.
func (d *Dao) GetLanguages() []*mtproto.LangPackLanguage {
	return defaultLanguages
}

// GetLanguage returns a single language by code.
func (d *Dao) GetLanguage(langCode string) (*mtproto.LangPackLanguage, error) {
	for _, lang := range defaultLanguages {
		if lang.LangCode == langCode {
			return lang, nil
		}
	}
	return nil, fmt.Errorf("language not found: %s", langCode)
}

// GetLangPack returns the full language pack. If not cached, fetches from Telegram.
func (d *Dao) GetLangPack(ctx context.Context, platform, langCode string) (*LangPackEntry, error) {
	cacheKey := platform + "/" + langCode

	// Check memory cache first
	d.mu.RLock()
	entry, ok := d.cache[cacheKey]
	d.mu.RUnlock()
	if ok {
		return entry, nil
	}

	// Check local file cache
	entry, err := d.loadFromFile(langCode, platform)
	if err == nil && entry != nil {
		// Merge custom strings before caching
		entry.Strings = append(entry.Strings, GetCustomStrings(langCode)...)
		d.mu.Lock()
		d.cache[cacheKey] = entry
		d.mu.Unlock()
		return entry, nil
	}

	// Fetch from Telegram translations
	entry, err = d.fetchAndCache(ctx, langCode, platform)
	if err != nil {
		return nil, err
	}

	return entry, nil
}

// GetStrings returns specific keys from the language pack.
func (d *Dao) GetStrings(ctx context.Context, platform, langCode string, keys []string) ([]*mtproto.LangPackString, error) {
	entry, err := d.GetLangPack(ctx, platform, langCode)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return entry.Strings, nil
	}

	keySet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		keySet[k] = struct{}{}
	}

	var result []*mtproto.LangPackString
	for _, s := range entry.Strings {
		if _, ok := keySet[s.Key]; ok {
			result = append(result, s)
		}
	}

	return result, nil
}

// fetchAndCache fetches the language pack from Telegram export URL, saves to file, and caches in memory.
func (d *Dao) fetchAndCache(ctx context.Context, langCode, platform string) (*LangPackEntry, error) {
	log := logx.WithContext(ctx)

	// Map platform to Telegram export path
	telegramPlatform := mapPlatform(platform)

	// https://translations.telegram.org/{langCode}/{platform}/export
	exportURL := fmt.Sprintf("https://translations.telegram.org/%s/%s/export", langCode, telegramPlatform)
	log.Infof("fetchAndCache - fetching langpack from %s", exportURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exportURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Teamgram-Server/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch langpack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch langpack: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// Save to local file
	filePath := d.filePath(langCode, platform)
	os.MkdirAll(filepath.Dir(filePath), 0755)
	if err := os.WriteFile(filePath, body, 0644); err != nil {
		log.Errorf("fetchAndCache - failed to save file %s: %v", filePath, err)
		// Continue even if save fails
	} else {
		log.Infof("fetchAndCache - saved langpack to %s (%d bytes)", filePath, len(body))
	}

	// Parse the .strings content
	strings, err := parseAppleStrings(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse strings file: %w", err)
	}

	entry := &LangPackEntry{
		Strings:  append(strings, GetCustomStrings(langCode)...),
		Version:  langPackVersion,
		LoadedAt: time.Now(),
	}

	cacheKey := platform + "/" + langCode
	d.mu.Lock()
	d.cache[cacheKey] = entry
	d.mu.Unlock()

	log.Infof("fetchAndCache - loaded %d strings for %s/%s", len(strings), platform, langCode)
	return entry, nil
}

// loadFromFile loads a cached .strings file from disk.
func (d *Dao) loadFromFile(langCode, platform string) (*LangPackEntry, error) {
	filePath := d.filePath(langCode, platform)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	strings, err := parseAppleStrings(string(data))
	if err != nil {
		return nil, err
	}

	return &LangPackEntry{
		Strings:  strings,
		Version:  langPackVersion,
		LoadedAt: time.Now(),
	}, nil
}

func (d *Dao) filePath(langCode, platform string) string {
	return filepath.Join(d.cacheDir, platform, langCode+".strings")
}

// mapPlatform maps client lang_pack values to Telegram translation platform names.
func mapPlatform(platform string) string {
	switch strings.ToLower(platform) {
	case "ios", "macos":
		return "ios"
	case "android", "android_x":
		return "android"
	case "tdesktop", "desktop":
		return "tdesktop"
	default:
		return "ios"
	}
}

// parseAppleStrings parses Apple .strings format: "key" = "value";
func parseAppleStrings(content string) ([]*mtproto.LangPackString, error) {
	var result []*mtproto.LangPackString

	scanner := bufio.NewScanner(strings.NewReader(content))
	// Increase buffer for long lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		// Parse "key" = "value";
		if !strings.HasPrefix(line, "\"") {
			continue
		}

		key, value, ok := parseStringsLine(line)
		if !ok {
			continue
		}

		result = append(result, mtproto.MakeTLLangPackString(&mtproto.LangPackString{
			Key:   key,
			Value: value,
		}).To_LangPackString())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// parseStringsLine parses a single "key" = "value"; line.
func parseStringsLine(line string) (key, value string, ok bool) {
	// Find key between first pair of quotes
	if len(line) < 5 || line[0] != '"' {
		return "", "", false
	}

	keyEnd := strings.Index(line[1:], "\"")
	if keyEnd < 0 {
		return "", "", false
	}
	key = line[1 : keyEnd+1]

	// Find " = " separator
	rest := line[keyEnd+2:]
	eqIdx := strings.Index(rest, "= \"")
	if eqIdx < 0 {
		return "", "", false
	}

	// Extract value - everything between "= "" and the trailing ";"
	valueStart := eqIdx + 3
	valueContent := rest[valueStart:]

	// Find the closing "; (could contain escaped quotes)
	valueEnd := strings.LastIndex(valueContent, "\";")
	if valueEnd < 0 {
		// Try just ending with "
		valueEnd = strings.LastIndex(valueContent, "\"")
		if valueEnd < 0 {
			return "", "", false
		}
	}
	value = valueContent[:valueEnd]

	// Unescape common escape sequences
	value = strings.ReplaceAll(value, "\\n", "\n")
	value = strings.ReplaceAll(value, "\\\"", "\"")
	value = strings.ReplaceAll(value, "\\\\", "\\")

	return key, value, true
}
