package core

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dao"
)

// MessagesGetStickerSet handles the messages.getStickerSet TL method.
func (c *StickersCore) MessagesGetStickerSet(in *mtproto.TLMessagesGetStickerSet) (*mtproto.Messages_StickerSet, error) {
	var shortName string

	stickerSet := in.GetStickerset()
	if stickerSet == nil {
		c.Logger.Errorf("messages.getStickerSet - nil stickerset")
		return nil, mtproto.ErrStickerIdInvalid
	}

	switch stickerSet.GetPredicateName() {
	case mtproto.Predicate_inputStickerSetShortName:
		shortName = stickerSet.GetShortName()
	case mtproto.Predicate_inputStickerSetID:
		setDO, err := c.svcCtx.Dao.StickerSetsDAO.SelectBySetId(c.ctx, stickerSet.GetId())
		if err != nil {
			c.Logger.Errorf("messages.getStickerSet - SelectBySetId(%d) error: %v", stickerSet.GetId(), err)
			return nil, mtproto.ErrStickerIdInvalid
		}
		if setDO == nil {
			return nil, mtproto.ErrStickerIdInvalid
		}
		return c.buildStickerSetFromCache(setDO)
	default:
		c.Logger.Errorf("messages.getStickerSet - unsupported predicate: %s", stickerSet.GetPredicateName())
		return nil, mtproto.ErrStickerIdInvalid
	}

	if shortName == "" {
		return nil, mtproto.ErrStickerIdInvalid
	}

	// 1. Check DB cache
	setDO, err := c.svcCtx.Dao.StickerSetsDAO.SelectByShortName(c.ctx, shortName)
	if err != nil {
		c.Logger.Errorf("messages.getStickerSet - SelectByShortName(%s) error: %v", shortName, err)
		return nil, mtproto.ErrInternelServerError
	}

	if setDO != nil {
		return c.buildStickerSetFromCache(setDO)
	}

	// 2. Not cached — fetch from Bot API and download all files synchronously
	return c.fetchAndCacheStickerSet(shortName)
}

// buildStickerSetFromCache reconstructs the Messages_StickerSet from cached DB data.
func (c *StickersCore) buildStickerSetFromCache(setDO *dataobject.StickerSetsDO) (*mtproto.Messages_StickerSet, error) {
	docDOs, err := c.svcCtx.Dao.StickerSetDocumentsDAO.SelectBySetId(c.ctx, setDO.SetId)
	if err != nil {
		c.Logger.Errorf("buildStickerSetFromCache - SelectBySetId error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	documents := make([]*mtproto.Document, 0, len(docDOs))
	for i := range docDOs {
		doc, err := dao.DeserializeStickerDoc(docDOs[i].DocumentData)
		if err != nil {
			c.Logger.Errorf("buildStickerSetFromCache - deserialize document %d error: %v", docDOs[i].DocumentId, err)
			continue
		}
		documents = append(documents, doc)
	}

	packs := buildStickerPacks(docDOs)
	stickerSet := makeStickerSetFromDO(setDO)

	return mtproto.MakeTLMessagesStickerSet(&mtproto.Messages_StickerSet{
		Set:       stickerSet,
		Packs:     packs,
		Keywords:  []*mtproto.StickerKeyword{},
		Documents: documents,
	}).To_Messages_StickerSet(), nil
}

// fetchAndCacheStickerSet fetches a sticker set from Telegram Bot API, downloads all files
// to DFS synchronously, saves everything to DB, and returns the response.
func (c *StickersCore) fetchAndCacheStickerSet(shortName string) (*mtproto.Messages_StickerSet, error) {
	botResult, err := c.svcCtx.Dao.BotAPI.GetStickerSet(c.ctx, shortName)
	if err != nil {
		c.Logger.Errorf("fetchAndCacheStickerSet - BotAPI.GetStickerSet(%s) error: %v", shortName, err)
		return nil, mtproto.ErrStickerIdInvalid
	}

	// Generate set IDs
	setId := c.svcCtx.Dao.IDGenClient2.NextId(c.ctx)
	setAccessHash := rand.Int63()
	now := time.Now().Unix()

	dataJson, _ := json.Marshal(botResult)

	// Build download inputs for each sticker
	inputs := make([]dao.StickerDownloadInput, 0, len(botResult.Stickers))
	for _, sticker := range botResult.Stickers {
		inputs = append(inputs, dao.StickerDownloadInput{
			BotFileId:       sticker.FileId,
			BotFileUniqueId: sticker.FileUniqueId,
			MimeType:        stickerMimeType(sticker),
			Attributes:      buildDocumentAttributes(sticker, setId, setAccessHash),
		})
	}

	// Download all files and upload to DFS synchronously
	dfsDocs, err := c.svcCtx.Dao.DownloadAndUploadStickerFiles(c.ctx, inputs)
	if err != nil {
		c.Logger.Errorf("fetchAndCacheStickerSet - DownloadAndUploadStickerFiles(%s) error: %v", shortName, err)
		return nil, mtproto.ErrInternelServerError
	}

	// Build document DOs from DFS results (real DFS-assigned IDs)
	stickerDocDOs := make([]*dataobject.StickerSetDocumentsDO, 0, len(dfsDocs))
	for idx, dfsDoc := range dfsDocs {
		sticker := botResult.Stickers[idx]

		docData, err := dao.SerializeStickerDoc(dfsDoc)
		if err != nil {
			c.Logger.Errorf("fetchAndCacheStickerSet - serialize dfsDoc error: %v", err)
			docData = ""
		}

		thumbFileId := ""
		if sticker.Thumbnail != nil {
			thumbFileId = sticker.Thumbnail.FileId
		}

		stickerDocDOs = append(stickerDocDOs, &dataobject.StickerSetDocumentsDO{
			SetId:           setId,
			DocumentId:      dfsDoc.GetId(),
			StickerIndex:    int32(idx),
			Emoji:           sticker.Emoji,
			BotFileId:       sticker.FileId,
			BotFileUniqueId: sticker.FileUniqueId,
			BotThumbFileId:  thumbFileId,
			DocumentData:    docData,
			FileDownloaded:  true,
		})
	}

	// Determine set flags
	isAnimated := len(botResult.Stickers) > 0 && botResult.Stickers[0].IsAnimated
	isVideo := len(botResult.Stickers) > 0 && botResult.Stickers[0].IsVideo
	isMasks := botResult.StickerType == "mask"
	isEmojis := botResult.StickerType == "custom_emoji"

	setDO := &dataobject.StickerSetsDO{
		SetId:        setId,
		AccessHash:   setAccessHash,
		ShortName:    shortName,
		Title:        botResult.Title,
		StickerType:  botResult.StickerType,
		IsAnimated:   isAnimated,
		IsVideo:      isVideo,
		IsMasks:      isMasks,
		IsEmojis:     isEmojis,
		IsOfficial:   false,
		StickerCount: int32(len(botResult.Stickers)),
		Hash:         0,
		ThumbDocId:   0,
		DataJson:     string(dataJson),
		FetchedAt:    now,
	}

	_, rowsAffected, err := c.svcCtx.Dao.StickerSetsDAO.InsertIgnore(c.ctx, setDO)
	if err != nil {
		c.Logger.Errorf("fetchAndCacheStickerSet - InsertIgnore sticker_sets error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	// Another concurrent request already inserted this set — fall back to cached data
	if rowsAffected == 0 {
		c.Logger.Infof("fetchAndCacheStickerSet - set %s already cached by another request, falling back", shortName)
		cachedDO, err2 := c.svcCtx.Dao.StickerSetsDAO.SelectByShortName(c.ctx, shortName)
		if err2 != nil || cachedDO == nil {
			c.Logger.Errorf("fetchAndCacheStickerSet - fallback SelectByShortName(%s) error: %v", shortName, err2)
			return nil, mtproto.ErrInternelServerError
		}
		return c.buildStickerSetFromCache(cachedDO)
	}

	for _, docDO := range stickerDocDOs {
		_, _, err = c.svcCtx.Dao.StickerSetDocumentsDAO.InsertIgnore(c.ctx, docDO)
		if err != nil {
			c.Logger.Errorf("fetchAndCacheStickerSet - InsertIgnore sticker_set_documents error: %v", err)
		}
	}

	packs := buildStickerPacks2(stickerDocDOs)
	stickerSetPB := makeStickerSetFromDO(setDO)

	return mtproto.MakeTLMessagesStickerSet(&mtproto.Messages_StickerSet{
		Set:       stickerSetPB,
		Packs:     packs,
		Keywords:  []*mtproto.StickerKeyword{},
		Documents: dfsDocs,
	}).To_Messages_StickerSet(), nil
}

// --- Helper functions ---

func stickerMimeType(s dao.BotAPISticker) string {
	if s.IsAnimated {
		return "application/x-tgsticker"
	}
	if s.IsVideo {
		return "video/webm"
	}
	return "image/webp"
}

func stickerExt(s dao.BotAPISticker) string {
	if s.IsAnimated {
		return ".tgs"
	}
	if s.IsVideo {
		return ".webm"
	}
	return ".webp"
}

func buildDocumentAttributes(s dao.BotAPISticker, setId, setAccessHash int64) []*mtproto.DocumentAttribute {
	attrs := make([]*mtproto.DocumentAttribute, 0, 3)

	attrs = append(attrs, mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{
		Alt: s.Emoji,
		Stickerset: mtproto.MakeTLInputStickerSetID(&mtproto.InputStickerSet{
			Id:         setId,
			AccessHash: setAccessHash,
		}).To_InputStickerSet(),
	}).To_DocumentAttribute())

	attrs = append(attrs, mtproto.MakeTLDocumentAttributeImageSize(&mtproto.DocumentAttribute{
		W: s.Width,
		H: s.Height,
	}).To_DocumentAttribute())

	attrs = append(attrs, mtproto.MakeTLDocumentAttributeFilename(&mtproto.DocumentAttribute{
		FileName: s.FileUniqueId + stickerExt(s),
	}).To_DocumentAttribute())

	return attrs
}

func buildStickerPacks(docDOs []dataobject.StickerSetDocumentsDO) []*mtproto.StickerPack {
	emojiMap := make(map[string][]int64)
	for _, d := range docDOs {
		if d.Emoji != "" {
			emojiMap[d.Emoji] = append(emojiMap[d.Emoji], d.DocumentId)
		}
	}

	packs := make([]*mtproto.StickerPack, 0, len(emojiMap))
	for emoji, docIds := range emojiMap {
		packs = append(packs, mtproto.MakeTLStickerPack(&mtproto.StickerPack{
			Emoticon:  emoji,
			Documents: docIds,
		}).To_StickerPack())
	}
	return packs
}

func buildStickerPacks2(docDOs []*dataobject.StickerSetDocumentsDO) []*mtproto.StickerPack {
	emojiMap := make(map[string][]int64)
	for _, d := range docDOs {
		if d.Emoji != "" {
			emojiMap[d.Emoji] = append(emojiMap[d.Emoji], d.DocumentId)
		}
	}

	packs := make([]*mtproto.StickerPack, 0, len(emojiMap))
	for emoji, docIds := range emojiMap {
		packs = append(packs, mtproto.MakeTLStickerPack(&mtproto.StickerPack{
			Emoticon:  emoji,
			Documents: docIds,
		}).To_StickerPack())
	}
	return packs
}

func makeStickerSetFromDO(setDO *dataobject.StickerSetsDO) *mtproto.StickerSet {
	ss := &mtproto.StickerSet{
		Id:         setDO.SetId,
		AccessHash: setDO.AccessHash,
		Title:      setDO.Title,
		ShortName:  setDO.ShortName,
		Count:      setDO.StickerCount,
		Hash:       setDO.Hash,
		Animated:   setDO.IsAnimated,
		Videos:     setDO.IsVideo,
		Masks:      setDO.IsMasks,
		Emojis:     setDO.IsEmojis,
		Official:   setDO.IsOfficial,
	}

	if setDO.ThumbDocId != 0 {
		ss.ThumbDocumentId = &types.Int64Value{Value: setDO.ThumbDocId}
	}

	return mtproto.MakeTLStickerSet(ss).To_StickerSet()
}
