package core

import (
	"github.com/teamgram/proto/mtproto"
)

// LangpackGetLanguages
// langpack.getLanguages lang_pack:string = Vector<LangPackLanguage>;
func (c *LangpackCore) LangpackGetLanguages(in *mtproto.TLLangpackGetLanguages) (*mtproto.Vector_LangPackLanguage, error) {
	languages := c.svcCtx.Dao.GetLanguages()
	c.Logger.Infof("langpack.getLanguages - returning %d languages", len(languages))

	return &mtproto.Vector_LangPackLanguage{
		Datas: languages,
	}, nil
}
