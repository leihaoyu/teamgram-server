package core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dao"
	"github.com/zeromicro/go-zero/core/logx"
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
		// Look up from DB by set_id — if found, return directly from cache (no second query)
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
		// Found in cache — build response from DB
		return c.buildStickerSetFromCache(setDO)
	}

	// 2. Not cached — fetch from Bot API
	return c.fetchAndCacheStickerSet(shortName)
}

// buildStickerSetFromCache reconstructs the Messages_StickerSet from cached DB data.
func (c *StickersCore) buildStickerSetFromCache(setDO *dataobject.StickerSetsDO) (*mtproto.Messages_StickerSet, error) {
	// Load sticker documents from our mapping table
	docDOs, err := c.svcCtx.Dao.StickerSetDocumentsDAO.SelectBySetId(c.ctx, setDO.SetId)
	if err != nil {
		c.Logger.Errorf("buildStickerSetFromCache - SelectBySetId error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	// Reconstruct Document protobufs from stored data
	documents := make([]*mtproto.Document, 0, len(docDOs))
	for i := range docDOs {
		doc, err := deserializeDocument(docDOs[i].DocumentData)
		if err != nil {
			c.Logger.Errorf("buildStickerSetFromCache - deserialize document %d error: %v", docDOs[i].DocumentId, err)
			continue
		}
		documents = append(documents, doc)
	}

	// Build StickerPack list (emoji -> []document_id)
	packs := buildStickerPacks(docDOs)

	// Build StickerSet protobuf
	stickerSet := makeStickerSetFromDO(setDO)

	return mtproto.MakeTLMessagesStickerSet(&mtproto.Messages_StickerSet{
		Set:       stickerSet,
		Packs:     packs,
		Keywords:  []*mtproto.StickerKeyword{},
		Documents: documents,
	}).To_Messages_StickerSet(), nil
}

// fetchAndCacheStickerSet fetches a sticker set from Telegram Bot API, saves it to DB,
// kicks off async file downloads, and returns the response.
func (c *StickersCore) fetchAndCacheStickerSet(shortName string) (*mtproto.Messages_StickerSet, error) {
	botResult, err := c.svcCtx.Dao.BotAPI.GetStickerSet(c.ctx, shortName)
	if err != nil {
		c.Logger.Errorf("fetchAndCacheStickerSet - BotAPI.GetStickerSet(%s) error: %v", shortName, err)
		return nil, mtproto.ErrStickerIdInvalid
	}

	// Generate our IDs
	setId := c.svcCtx.Dao.IDGenClient2.NextId(c.ctx)
	setAccessHash := rand.Int63()
	now := time.Now().Unix()

	// Save the raw JSON for debugging
	dataJson, _ := json.Marshal(botResult)

	// Process each sticker
	documents := make([]*mtproto.Document, 0, len(botResult.Stickers))
	stickerDocDOs := make([]*dataobject.StickerSetDocumentsDO, 0, len(botResult.Stickers))

	for idx, sticker := range botResult.Stickers {
		docId := c.svcCtx.Dao.IDGenClient2.NextId(c.ctx)
		docAccessHash := generateAccessHash(sticker)
		mimeType := stickerMimeType(sticker)
		fileSize := sticker.FileSize
		if fileSize == 0 {
			fileSize = 1 // avoid zero size
		}

		// Build document attributes
		attributes := buildDocumentAttributes(sticker, setId, setAccessHash)

		// Build the Document protobuf
		doc := mtproto.MakeTLDocument(&mtproto.Document{
			Id:            docId,
			AccessHash:    docAccessHash,
			FileReference: []byte{},
			Date:          int32(now),
			MimeType:      mimeType,
			Size2_INT32:   int32(fileSize),
			Size2_INT64:   fileSize,
			Thumbs:        buildStickerThumbs(sticker),
			VideoThumbs:   nil,
			DcId:          1,
			Attributes:    attributes,
		}).To_Document()

		documents = append(documents, doc)

		// Serialize the Document protobuf for DB storage
		docData, err := serializeDocument(doc)
		if err != nil {
			c.Logger.Errorf("fetchAndCacheStickerSet - serialize document error: %v", err)
			docData = ""
		}

		thumbFileId := ""
		if sticker.Thumbnail != nil {
			thumbFileId = sticker.Thumbnail.FileId
		}

		stickerDocDOs = append(stickerDocDOs, &dataobject.StickerSetDocumentsDO{
			SetId:           setId,
			DocumentId:      docId,
			StickerIndex:    int32(idx),
			Emoji:           sticker.Emoji,
			BotFileId:       sticker.FileId,
			BotFileUniqueId: sticker.FileUniqueId,
			BotThumbFileId:  thumbFileId,
			DocumentData:    docData,
			FileDownloaded:  false,
		})
	}

	// Determine set flags
	isAnimated := len(botResult.Stickers) > 0 && botResult.Stickers[0].IsAnimated
	isVideo := len(botResult.Stickers) > 0 && botResult.Stickers[0].IsVideo
	isMasks := botResult.StickerType == "mask"
	isEmojis := botResult.StickerType == "custom_emoji"

	// Save sticker set to DB
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

	// Save individual sticker document mappings
	for _, docDO := range stickerDocDOs {
		_, _, err = c.svcCtx.Dao.StickerSetDocumentsDAO.InsertIgnore(c.ctx, docDO)
		if err != nil {
			c.Logger.Errorf("fetchAndCacheStickerSet - InsertIgnore sticker_set_documents error: %v", err)
		}
	}

	// Build packs
	packs := buildStickerPacks2(stickerDocDOs)

	// Build StickerSet protobuf
	stickerSetPB := makeStickerSetFromDO(setDO)

	// Kick off async file downloads with timeout
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("DownloadStickerFiles panic: %v", r)
			}
		}()
		dlCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		c.svcCtx.Dao.DownloadStickerFiles(dlCtx, setId)
	}()

	return mtproto.MakeTLMessagesStickerSet(&mtproto.Messages_StickerSet{
		Set:       stickerSetPB,
		Packs:     packs,
		Keywords:  []*mtproto.StickerKeyword{},
		Documents: documents,
	}).To_Messages_StickerSet(), nil
}

// --- Serialization helpers ---

func serializeDocument(doc *mtproto.Document) (string, error) {
	data, err := proto.Marshal(doc)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func deserializeDocument(s string) (*mtproto.Document, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	doc := &mtproto.Document{}
	if err := proto.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	return doc, nil
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

func generateAccessHash(s dao.BotAPISticker) int64 {
	// Follow the same pattern as DFS: int64(storageType)<<32 | int64(rand.Uint32())
	var storageType int64
	if s.IsAnimated {
		storageType = 5 // application type
	} else if s.IsVideo {
		storageType = 3 // video type
	} else {
		storageType = 1 // image type
	}
	return storageType<<32 | int64(rand.Uint32())
}

func buildDocumentAttributes(s dao.BotAPISticker, setId, setAccessHash int64) []*mtproto.DocumentAttribute {
	attrs := make([]*mtproto.DocumentAttribute, 0, 3)

	// documentAttributeSticker
	attrs = append(attrs, mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{
		Alt: s.Emoji,
		Stickerset: mtproto.MakeTLInputStickerSetID(&mtproto.InputStickerSet{
			Id:         setId,
			AccessHash: setAccessHash,
		}).To_InputStickerSet(),
	}).To_DocumentAttribute())

	// documentAttributeImageSize
	attrs = append(attrs, mtproto.MakeTLDocumentAttributeImageSize(&mtproto.DocumentAttribute{
		W: s.Width,
		H: s.Height,
	}).To_DocumentAttribute())

	// documentAttributeFilename
	attrs = append(attrs, mtproto.MakeTLDocumentAttributeFilename(&mtproto.DocumentAttribute{
		FileName: s.FileUniqueId + stickerExt(s),
	}).To_DocumentAttribute())

	return attrs
}

func buildStickerThumbs(s dao.BotAPISticker) []*mtproto.PhotoSize {
	if s.Thumbnail == nil {
		return nil
	}

	return []*mtproto.PhotoSize{
		mtproto.MakeTLPhotoSize(&mtproto.PhotoSize{
			Type:  "m",
			W:     s.Thumbnail.Width,
			H:     s.Thumbnail.Height,
			Size2: int32(s.Thumbnail.FileSize),
		}).To_PhotoSize(),
	}
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
