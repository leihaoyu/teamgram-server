package dao

import (
	"github.com/gogo/protobuf/types"
	"github.com/teamgram/proto/mtproto"
)

// defaultLanguages is the full list of supported languages.
// Based on Telegram's official langpack.getLanguages response, plus zh-hans and zh-hant.
var defaultLanguages = []*mtproto.LangPackLanguage{
	makeLang("English", "English", "en", "", "en", 11302, 11302, true, false, false),
	makeLang("Arabic", "العربية", "ar", "", "ar", 11302, 11227, true, false, true),
	makeLang("Belarusian", "Беларуская", "be", "", "be", 11302, 9629, true, false, false),
	makeLang("Catalan", "Català", "ca", "", "ca", 11302, 11302, true, false, false),
	makeLang("Croatian", "Hrvatski", "hr", "", "hr", 11302, 11302, true, false, false),
	makeLang("Czech", "Čeština", "cs", "", "cs", 11302, 11302, true, false, false),
	makeLang("Dutch", "Nederlands", "nl", "", "nl", 11302, 11302, true, false, false),
	makeLang("Finnish", "Suomi", "fi", "", "fi", 11302, 11302, true, false, false),
	makeLang("French", "Français", "fr", "", "fr", 11302, 11302, true, false, false),
	makeLang("German", "Deutsch", "de", "", "de", 11302, 10451, true, false, false),
	makeLang("Hebrew", "עברית", "he", "", "he", 11302, 11298, true, false, true),
	makeLang("Hungarian", "Magyar", "hu", "", "hu", 11302, 11176, true, false, false),
	makeLang("Indonesian", "Bahasa Indonesia", "id", "", "id", 11302, 11273, true, false, false),
	makeLang("Italian", "Italiano", "it", "", "it", 11302, 11302, true, false, false),
	makeLang("Kazakh", "Қазақша", "kk", "", "kk", 11302, 11302, true, false, false),
	makeLang("Korean", "한국어", "ko", "", "ko", 11302, 8136, true, false, false),
	makeLang("Malay", "Bahasa Melayu", "ms", "", "ms", 11302, 11302, true, false, false),
	makeLang("Norwegian (Bokmål)", "Norsk (Bokmål)", "nb", "", "nb", 11302, 11302, true, false, false),
	makeLang("Persian", "فارسی", "fa", "", "fa", 11302, 10953, true, false, true),
	makeLang("Polish", "Polski", "pl", "", "pl", 11302, 10674, true, false, false),
	makeLang("Portuguese (Brazil)", "Português (Brasil)", "pt-br", "", "pt", 11302, 11302, true, false, false),
	makeLang("Romanian", "Română", "ro", "", "ro", 11302, 11302, true, false, false),
	makeLang("Russian", "Русский", "ru", "", "ru", 11302, 11302, true, false, false),
	makeLang("Serbian", "Српски", "sr", "", "sr", 11302, 11302, true, false, false),
	makeLang("Slovak", "Slovenčina", "sk", "", "sk", 11302, 10638, true, false, false),
	makeLang("Spanish", "Español", "es", "", "es", 11302, 11302, true, false, false),
	makeLang("Swedish", "Svenska", "sv", "", "sv", 11302, 11302, true, false, false),
	makeLang("Turkish", "Türkçe", "tr", "", "tr", 11302, 11302, true, false, false),
	makeLang("Ukrainian", "Українська", "uk", "", "uk", 11302, 11302, true, false, false),
	makeLang("Uzbek", "Oʻzbek", "uz", "", "uz", 11302, 10061, true, false, false),
	makeLang("Vietnamese", "Tiếng Việt", "vi", "", "vi", 11302, 11022, true, false, false),
	// Added: Simplified Chinese and Traditional Chinese
	makeLang("Chinese (Simplified)", "简体中文", "zh-hans", "", "zh", 11302, 11302, true, false, false),
	makeLang("Chinese (Traditional)", "繁體中文", "zh-hant", "", "zh", 11302, 11302, true, false, false),
}

func makeLang(name, nativeName, langCode, baseLangCode, pluralCode string, stringsCount, translatedCount int32, official, beta, rtl bool) *mtproto.LangPackLanguage {
	lang := &mtproto.LangPackLanguage{
		Official:        official,
		Rtl:             rtl,
		Beta:            beta,
		Name:            name,
		NativeName:      nativeName,
		LangCode:        langCode,
		PluralCode:      pluralCode,
		StringsCount:    stringsCount,
		TranslatedCount: translatedCount,
		TranslationsUrl: "https://translations.telegram.org/" + langCode + "/",
	}
	if baseLangCode != "" {
		lang.BaseLangCode = &types.StringValue{Value: baseLangCode}
	}
	return mtproto.MakeTLLangPackLanguage(lang).To_LangPackLanguage()
}
