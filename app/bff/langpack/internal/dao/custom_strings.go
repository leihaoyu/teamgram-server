package dao

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"

	"github.com/teamgram/proto/mtproto"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	customStrings     map[string]map[string]string
	customStringsOnce sync.Once
)

//go:embed custom_strings.json
var customStringsJSON []byte

func loadCustomStrings() {
	customStrings = make(map[string]map[string]string)

	if err := json.Unmarshal(customStringsJSON, &customStrings); err != nil {
		logx.Errorf("loadCustomStrings - failed to parse embedded JSON: %v", err)
		return
	}

	logx.Infof("loadCustomStrings - loaded %d languages from embedded custom_strings.json", len(customStrings))
}

// GetCustomStrings returns custom localization strings for the given language code.
// Falls back to English if the language is not found.
func GetCustomStrings(langCode string) []*mtproto.LangPackString {
	customStringsOnce.Do(loadCustomStrings)

	code := strings.ToLower(langCode)

	strs, ok := customStrings[code]
	if !ok {
		// Try base language prefix (e.g. "pt-br" -> "pt", "zh-hans" -> "zh")
		if idx := strings.IndexAny(code, "-_"); idx > 0 {
			strs, ok = customStrings[code[:idx]]
		}
		if !ok {
			// Fallback to English
			strs = customStrings["en"]
		}
	}

	if len(strs) == 0 {
		return nil
	}

	result := make([]*mtproto.LangPackString, 0, len(strs))
	for key, value := range strs {
		result = append(result, mtproto.MakeTLLangPackString(&mtproto.LangPackString{
			Key:   key,
			Value: value,
		}).To_LangPackString())
	}

	return result
}
