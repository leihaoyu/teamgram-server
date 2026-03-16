package core

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dao"
)

// MessagesGetStickers returns Documents matching the given emoji across the user's installed sticker sets.
func (c *StickersCore) MessagesGetStickers(in *mtproto.TLMessagesGetStickers) (*mtproto.Messages_Stickers, error) {
	userId := c.MD.UserId
	emoticon := in.Emoticon

	if emoticon == "" {
		return mtproto.MakeTLMessagesStickers(&mtproto.Messages_Stickers{
			Hash:     0,
			Stickers: []*mtproto.Document{},
		}).To_Messages_Stickers(), nil
	}

	// 1. Get user's installed set_ids (regular type = 0)
	installedRows, err := c.svcCtx.Dao.UserInstalledStickerSetsDAO.SelectByUserAndType(c.ctx, userId, 0)
	if err != nil {
		c.Logger.Errorf("messages.getStickers - SelectByUserAndType(%d) error: %v", userId, err)
		return nil, mtproto.ErrInternelServerError
	}

	if len(installedRows) == 0 {
		return mtproto.MakeTLMessagesStickers(&mtproto.Messages_Stickers{
			Hash:     0,
			Stickers: []*mtproto.Document{},
		}).To_Messages_Stickers(), nil
	}

	setIds := make([]int64, len(installedRows))
	for i, r := range installedRows {
		setIds[i] = r.SetId
	}

	// 2. Query documents matching emoji across all installed sets
	docDOs, err := c.svcCtx.Dao.StickerSetDocumentsDAO.SelectBySetIdsAndEmoji(c.ctx, setIds, emoticon)
	if err != nil {
		c.Logger.Errorf("messages.getStickers - SelectBySetIdsAndEmoji error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	if len(docDOs) == 0 {
		return mtproto.MakeTLMessagesStickers(&mtproto.Messages_Stickers{
			Hash:     0,
			Stickers: []*mtproto.Document{},
		}).To_Messages_Stickers(), nil
	}

	// 3. Deserialize documents and compute hash
	stickers := make([]*mtproto.Document, 0, len(docDOs))
	var hashAcc uint64
	for i := range docDOs {
		doc, err2 := dao.DeserializeStickerDoc(docDOs[i].DocumentData)
		if err2 != nil {
			c.Logger.Errorf("messages.getStickers - deserialize doc %d error: %v", docDOs[i].DocumentId, err2)
			continue
		}
		stickers = append(stickers, doc)
		telegramCombineInt64Hash(&hashAcc, uint64(doc.GetId()))
	}

	hash := int64(hashAcc)

	// 4. Check NotModified
	if in.Hash != 0 && in.Hash == hash {
		return mtproto.MakeTLMessagesStickersNotModified(nil).To_Messages_Stickers(), nil
	}

	return mtproto.MakeTLMessagesStickers(&mtproto.Messages_Stickers{
		Hash:     hash,
		Stickers: stickers,
	}).To_Messages_Stickers(), nil
}
