package core

import (
	"github.com/teamgram/proto/mtproto"
)

// LangpackGetLangPack
// langpack.getLangPack lang_pack:string lang_code:string = LangPackDifference;
func (c *LangpackCore) LangpackGetLangPack(in *mtproto.TLLangpackGetLangPack) (*mtproto.LangPackDifference, error) {
	langCode := in.GetLangCode()
	platform := in.GetLangPack()

	if langCode == "" {
		langCode = "en"
	}

	entry, err := c.svcCtx.Dao.GetLangPack(c.ctx, platform, langCode)
	if err != nil {
		c.Logger.Errorf("langpack.getLangPack - platform=%s langCode=%s error: %v", platform, langCode, err)
		return nil, err
	}

	c.Logger.Infof("langpack.getLangPack - platform=%s langCode=%s strings=%d version=%d",
		platform, langCode, len(entry.Strings), entry.Version)

	return mtproto.MakeTLLangPackDifference(&mtproto.LangPackDifference{
		LangCode:    langCode,
		FromVersion: 0,
		Version:     entry.Version,
		Strings:     entry.Strings,
	}).To_LangPackDifference(), nil
}
