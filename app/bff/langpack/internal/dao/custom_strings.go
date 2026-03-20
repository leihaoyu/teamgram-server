package dao

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/teamgram/proto/mtproto"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	customStrings     map[string]map[string]string
	customStringsOnce sync.Once
)

// customStringsPath is the path to the JSON file containing custom localization strings.
// Can be overridden before first call to GetCustomStrings.
var customStringsPath = "data/langpack/custom_strings.json"

func loadCustomStrings() {
	customStrings = make(map[string]map[string]string)

	data, err := os.ReadFile(customStringsPath)
	if err != nil {
		logx.Errorf("loadCustomStrings - failed to read %s: %v", customStringsPath, err)
		return
	}

	if err := json.Unmarshal(data, &customStrings); err != nil {
		logx.Errorf("loadCustomStrings - failed to parse %s: %v", customStringsPath, err)
		return
	}

	logx.Infof("loadCustomStrings - loaded %d languages from %s", len(customStrings), customStringsPath)
}

// GetCustomStrings returns custom localization strings for the given language code.
// Falls back to English if the language is not found.
func GetCustomStrings(langCode string) []*mtproto.LangPackString {
	customStringsOnce.Do(loadCustomStrings)

	code := strings.ToLower(langCode)

	strs, ok := customStrings[code]
	if !ok {
		// Fallback to English
		strs = customStrings["en"]
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
