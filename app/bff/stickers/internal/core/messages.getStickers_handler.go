package core

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dao"
)

// magicEmoticonMap maps Telegram's "magic" compound emoticon queries to the real single emoji
// used for DB lookup. These are special queries from iOS/Android clients:
//   - "👋⭐️" → greeting stickers (empty chat wave animation)
//   - "⭐️⭐️" → premium sticker examples (premium UI showcase)
//   - "📂⭐️" → all premium stickers (sticker keyboard premium section)
var magicEmoticonMap = map[string]string{
	"👋⭐️": "👋",
	"⭐️⭐️": "",  // premium — return empty
	"📂⭐️": "",  // premium — return empty
}

var emptyStickersResult = mtproto.MakeTLMessagesStickers(&mtproto.Messages_Stickers{
	Hash:     0,
	Stickers: []*mtproto.Document{},
}).To_Messages_Stickers()

// MessagesGetStickers returns Documents matching the given emoji across the user's installed sticker sets,
// or across all cached sets for special "magic" emoticon queries.
func (c *StickersCore) MessagesGetStickers(in *mtproto.TLMessagesGetStickers) (*mtproto.Messages_Stickers, error) {
	userId := c.MD.UserId
	emoticon := in.Emoticon

	if emoticon == "" {
		return emptyStickersResult, nil
	}

	// Check for magic emoticon queries (greeting, premium, etc.)
	if realEmoji, ok := magicEmoticonMap[emoticon]; ok {
		if realEmoji == "" {
			// Premium-related queries — not supported, return empty
			return emptyStickersResult, nil
		}
		// Greeting stickers — search across ALL cached sets by the real emoji
		return c.getStickersGlobal(realEmoji, in.Hash)
	}

	// Normal path: search across user's installed sticker sets
	return c.getStickersInstalled(userId, emoticon, in.Hash)
}

// getStickersGlobal searches for stickers matching emoji across ALL cached sticker sets.
func (c *StickersCore) getStickersGlobal(emoji string, requestHash int64) (*mtproto.Messages_Stickers, error) {
	docDOs, err := c.svcCtx.Dao.StickerSetDocumentsDAO.SelectByEmoji(c.ctx, emoji)
	if err != nil {
		c.Logger.Errorf("messages.getStickers - SelectByEmoji(%s) error: %v", emoji, err)
		return nil, mtproto.ErrInternelServerError
	}

	if len(docDOs) == 0 {
		return emptyStickersResult, nil
	}

	return c.buildStickersResult(docDOs, requestHash)
}

// getStickersInstalled searches for stickers matching emoji across the user's installed sets.
func (c *StickersCore) getStickersInstalled(userId int64, emoticon string, requestHash int64) (*mtproto.Messages_Stickers, error) {
	installedRows, err := c.svcCtx.Dao.UserInstalledStickerSetsDAO.SelectByUserAndType(c.ctx, userId, 0)
	if err != nil {
		c.Logger.Errorf("messages.getStickers - SelectByUserAndType(%d) error: %v", userId, err)
		return nil, mtproto.ErrInternelServerError
	}

	if len(installedRows) == 0 {
		return emptyStickersResult, nil
	}

	setIds := make([]int64, len(installedRows))
	for i, r := range installedRows {
		setIds[i] = r.SetId
	}

	docDOs, err := c.svcCtx.Dao.StickerSetDocumentsDAO.SelectBySetIdsAndEmoji(c.ctx, setIds, emoticon)
	if err != nil {
		c.Logger.Errorf("messages.getStickers - SelectBySetIdsAndEmoji error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	if len(docDOs) == 0 {
		return emptyStickersResult, nil
	}

	return c.buildStickersResult(docDOs, requestHash)
}

// buildStickersResult deserializes document DOs, computes hash, and returns the result.
func (c *StickersCore) buildStickersResult(docDOs []dataobject.StickerSetDocumentsDO, requestHash int64) (*mtproto.Messages_Stickers, error) {
	stickers := make([]*mtproto.Document, 0, len(docDOs))
	var hashAcc uint64
	for i := range docDOs {
		doc, err := dao.DeserializeStickerDoc(docDOs[i].DocumentData)
		if err != nil {
			c.Logger.Errorf("messages.getStickers - deserialize doc %d error: %v", docDOs[i].DocumentId, err)
			continue
		}
		stickers = append(stickers, doc)
		telegramCombineInt64Hash(&hashAcc, uint64(doc.GetId()))
	}

	hash := int64(hashAcc)

	if requestHash != 0 && requestHash == hash {
		return mtproto.MakeTLMessagesStickersNotModified(nil).To_Messages_Stickers(), nil
	}

	return mtproto.MakeTLMessagesStickers(&mtproto.Messages_Stickers{
		Hash:     hash,
		Stickers: stickers,
	}).To_Messages_Stickers(), nil
}
