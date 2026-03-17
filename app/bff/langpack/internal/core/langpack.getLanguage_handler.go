package core

import (
	"github.com/teamgram/proto/mtproto"
)

// LangpackGetLanguage
// langpack.getLanguage lang_pack:string lang_code:string = LangPackLanguage;
func (c *LangpackCore) LangpackGetLanguage(in *mtproto.TLLangpackGetLanguage) (*mtproto.LangPackLanguage, error) {
	lang, err := c.svcCtx.Dao.GetLanguage(in.GetLangCode())
	if err != nil {
		c.Logger.Errorf("langpack.getLanguage - langCode=%s not found: %v", in.GetLangCode(), err)
		return nil, mtproto.ErrLangCodeNotSupported
	}

	c.Logger.Infof("langpack.getLanguage - langCode=%s name=%s", in.GetLangCode(), lang.Name)
	return lang, nil
}
