package dao

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/teamgram/proto/mtproto"
)

// langPackVersion is bumped when language packs are updated.
// Clients will re-fetch on version mismatch.
var langPackVersion = int32(6)

// LangPackEntry stores a parsed language pack.
type LangPackEntry struct {
	Strings []*mtproto.LangPackString
	Version int32
}

// Dao is the data access object for langpack.
type Dao struct {
	mu       sync.RWMutex
	cache    map[string]*LangPackEntry // key: langCode e.g. "en"
	cacheDir string
}

func New(_ interface{}) *Dao {
	// Resolve data directory relative to the executable so it works
	// regardless of the working directory (e.g. teamgramd/bin/).
	cacheDir := "data/langpack"
	if exe, err := os.Executable(); err == nil {
		// Try one level up first (Docker: /app/bin -> /app)
		// then two levels up (local dev: teamgramd/bin -> project root)
		for _, rel := range []string{"..", "../.."} {
			root := filepath.Join(filepath.Dir(exe), rel)
			candidate := filepath.Join(root, "data/langpack")
			if info, err2 := os.Stat(candidate); err2 == nil && info.IsDir() {
				cacheDir = candidate
				break
			}
		}
	}
	os.MkdirAll(cacheDir, 0755)

	return &Dao{
		cache:    make(map[string]*LangPackEntry),
		cacheDir: cacheDir,
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

// GetLangPack returns the full language pack from local files.
func (d *Dao) GetLangPack(ctx context.Context, _, langCode string) (*LangPackEntry, error) {
	cacheKey := langCode

	// Check memory cache first
	d.mu.RLock()
	entry, ok := d.cache[cacheKey]
	d.mu.RUnlock()
	if ok {
		return entry, nil
	}

	// Load from local file
	entry, err := d.loadFromFile(langCode)
	if err != nil {
		return nil, fmt.Errorf("langpack file not found for %s: %w", langCode, err)
	}

	d.mu.Lock()
	d.cache[cacheKey] = entry
	d.mu.Unlock()
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

// loadFromFile loads a .strings file from disk.
func (d *Dao) loadFromFile(langCode string) (*LangPackEntry, error) {
	filePath := d.filePath(langCode)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	strings, err := parseAppleStrings(string(data))
	if err != nil {
		return nil, err
	}

	return &LangPackEntry{
		Strings: strings,
		Version: langPackVersion,
	}, nil
}

func (d *Dao) filePath(langCode string) string {
	return filepath.Join(d.cacheDir, langCode+".strings")
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
