package core

import (
	"github.com/teamgram/proto/mtproto"
)

// AccountGetThemes - returns appearance themes for the settings page
func (c *ThemesCore) AccountGetThemes(in *mtproto.TLAccountGetThemes) (*mtproto.Account_Themes, error) {
	themes := defaultAppearanceThemes()

	return mtproto.MakeTLAccountThemes(&mtproto.Account_Themes{
		Hash:   int64(len(themes)),
		Themes: themes,
	}).To_Account_Themes(), nil
}

// AccountGetChatThemes - returns predefined emoticon-based chat themes
func (c *ThemesCore) AccountGetChatThemes(in *mtproto.TLAccountGetChatThemes) (*mtproto.Account_Themes, error) {
	themes := defaultChatThemes()

	return mtproto.MakeTLAccountThemes(&mtproto.Account_Themes{
		Hash:   int64(len(themes)),
		Themes: themes,
	}).To_Account_Themes(), nil
}

// AccountGetTheme
func (c *ThemesCore) AccountGetTheme(in *mtproto.TLAccountGetTheme) (*mtproto.Theme, error) {
	return nil, mtproto.ErrThemeInvalid
}

// AccountInstallTheme
func (c *ThemesCore) AccountInstallTheme(in *mtproto.TLAccountInstallTheme) (*mtproto.Bool, error) {
	return mtproto.BoolTrue, nil
}

// AccountSaveTheme
func (c *ThemesCore) AccountSaveTheme(in *mtproto.TLAccountSaveTheme) (*mtproto.Bool, error) {
	return mtproto.BoolTrue, nil
}

// AccountCreateTheme
func (c *ThemesCore) AccountCreateTheme(in *mtproto.TLAccountCreateTheme) (*mtproto.Theme, error) {
	return nil, mtproto.ErrThemeInvalid
}

// AccountUpdateTheme
func (c *ThemesCore) AccountUpdateTheme(in *mtproto.TLAccountUpdateTheme) (*mtproto.Theme, error) {
	return nil, mtproto.ErrThemeInvalid
}

// AccountUploadTheme
func (c *ThemesCore) AccountUploadTheme(in *mtproto.TLAccountUploadTheme) (*mtproto.Document, error) {
	return nil, mtproto.ErrThemeInvalid
}
