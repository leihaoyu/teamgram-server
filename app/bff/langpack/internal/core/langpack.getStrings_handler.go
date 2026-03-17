package core

import (
	"github.com/teamgram/proto/mtproto"
)

// LangpackGetStrings
// langpack.getStrings lang_pack:string lang_code:string keys:Vector<string> = Vector<LangPackString>;
func (c *LangpackCore) LangpackGetStrings(in *mtproto.TLLangpackGetStrings) (*mtproto.Vector_LangPackString, error) {
	langCode := in.GetLangCode()
	platform := in.GetLangPack()

	if langCode == "" {
		langCode = "en"
	}

	strings, err := c.svcCtx.Dao.GetStrings(c.ctx, platform, langCode, in.GetKeys())
	if err != nil {
		c.Logger.Errorf("langpack.getStrings - platform=%s langCode=%s keys=%v error: %v",
			platform, langCode, in.GetKeys(), err)
		return nil, err
	}

	c.Logger.Infof("langpack.getStrings - platform=%s langCode=%s requested=%d returned=%d",
		platform, langCode, len(in.GetKeys()), len(strings))

	return &mtproto.Vector_LangPackString{
		Datas: strings,
	}, nil
}
