package core

import (
	"github.com/teamgram/proto/mtproto"
)

// LangpackGetDifference
// langpack.getDifference lang_pack:string lang_code:string from_version:int = LangPackDifference;
func (c *LangpackCore) LangpackGetDifference(in *mtproto.TLLangpackGetDifference) (*mtproto.LangPackDifference, error) {
	langCode := in.GetLangCode()
	platform := in.GetLangPack()

	if langCode == "" {
		langCode = "en"
	}

	entry, err := c.svcCtx.Dao.GetLangPack(c.ctx, platform, langCode)
	if err != nil {
		c.Logger.Errorf("langpack.getDifference - platform=%s langCode=%s error: %v", platform, langCode, err)
		return nil, err
	}

	// If client already has the current version, return empty diff
	if in.GetFromVersion() >= entry.Version {
		c.Logger.Infof("langpack.getDifference - platform=%s langCode=%s fromVersion=%d >= currentVersion=%d, no diff",
			platform, langCode, in.GetFromVersion(), entry.Version)
		return mtproto.MakeTLLangPackDifference(&mtproto.LangPackDifference{
			LangCode:    langCode,
			FromVersion: in.GetFromVersion(),
			Version:     entry.Version,
			Strings:     []*mtproto.LangPackString{},
		}).To_LangPackDifference(), nil
	}

	// Otherwise return the full pack (we don't track incremental changes)
	c.Logger.Infof("langpack.getDifference - platform=%s langCode=%s fromVersion=%d currentVersion=%d returning %d strings",
		platform, langCode, in.GetFromVersion(), entry.Version, len(entry.Strings))

	return mtproto.MakeTLLangPackDifference(&mtproto.LangPackDifference{
		LangCode:    langCode,
		FromVersion: in.GetFromVersion(),
		Version:     entry.Version,
		Strings:     entry.Strings,
	}).To_LangPackDifference(), nil
}
