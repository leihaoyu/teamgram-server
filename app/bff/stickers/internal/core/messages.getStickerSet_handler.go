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
		return c.buildStickerSetFromCache(setDO, in.GetHash())
	case mtproto.Predicate_inputStickerSetAnimatedEmoji:
		shortName = "AnimatedEmojies"
	case mtproto.Predicate_inputStickerSetAnimatedEmojiAnimations:
		shortName = "EmojiAnimations"
	case mtproto.Predicate_inputStickerSetEmojiGenericAnimations:
		shortName = "EmojiGenericAnimations"
	case mtproto.Predicate_inputStickerSetEmojiDefaultStatuses:
		shortName = "StatusPack"
	case mtproto.Predicate_inputStickerSetEmojiDefaultTopicIcons:
		shortName = "Topics"
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
		return c.buildStickerSetFromCache(setDO, in.GetHash())
	}

	// 2. Not cached — fetch from Bot API and download all files synchronously
	result, err := c.fetchAndCacheStickerSet(shortName)
	if err != nil {
		// For system built-in sets, return an empty set instead of an error
		// so the client can silently handle it (instead of crashing on STICKER_ID_INVALID)
		if isSystemBuiltInPredicate(stickerSet.GetPredicateName()) {
			c.Logger.Infof("messages.getStickerSet - system set %s fetch failed, returning empty set: %v", shortName, err)
			return c.makeEmptyStickerSet(shortName), nil
		}
		return nil, err
	}
	return result, nil
}

// buildStickerSetFromCache reconstructs the Messages_StickerSet from cached DB data.
// requestHash is the client's cached hash; if non-zero and matching, returns stickerSetNotModified.
func (c *StickersCore) buildStickerSetFromCache(setDO *dataobject.StickerSetsDO, requestHash int32) (*mtproto.Messages_StickerSet, error) {
	docDOs, err := c.svcCtx.Dao.StickerSetDocumentsDAO.SelectBySetId(c.ctx, setDO.SetId)
	if err != nil {
		c.Logger.Errorf("buildStickerSetFromCache - SelectBySetId error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	// Compute hash from document IDs
	hash := computeStickerSetHash(docDOs)

	// NotModified: client already has the latest version
	if requestHash != 0 && requestHash == hash {
		return mtproto.MakeTLMessagesStickerSetNotModified(nil).To_Messages_StickerSet(), nil
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
	stickerSet.Hash = hash

	// Check if the current user has this set installed and set InstalledDate
	installRow, err := c.svcCtx.Dao.UserInstalledStickerSetsDAO.SelectByUserAndSetId(c.ctx, c.MD.UserId, setDO.SetId)
	if err != nil {
		c.Logger.Errorf("buildStickerSetFromCache - SelectByUserAndSetId error: %v", err)
	} else if installRow != nil {
		stickerSet.InstalledDate = &types.Int32Value{Value: int32(installRow.InstalledDate)}
	}

	return mtproto.MakeTLMessagesStickerSet(&mtproto.Messages_StickerSet{
		Set:       stickerSet,
		Packs:     packs,
		Keywords:  []*mtproto.StickerKeyword{},
		Documents: documents,
	}).To_Messages_StickerSet(), nil
}

// FetchAndCacheStickerSet is the exported entry point for fetching and caching a sticker set.
// It delegates to fetchAndCacheStickerSet which uses singleflight to deduplicate concurrent
// downloads of the same set.
func (c *StickersCore) FetchAndCacheStickerSet(shortName string) (*mtproto.Messages_StickerSet, error) {
	return c.fetchAndCacheStickerSet(shortName)
}

// fetchAndCacheStickerSet deduplicates concurrent downloads of the same sticker set using
// singleflight: if multiple goroutines request the same shortName simultaneously, only one
// download is performed and the result is shared.
func (c *StickersCore) fetchAndCacheStickerSet(shortName string) (*mtproto.Messages_StickerSet, error) {
	v, err, _ := c.svcCtx.Dao.FetchGroup.Do(shortName, func() (interface{}, error) {
		return c.doFetchAndCacheStickerSet(shortName)
	})
	if err != nil {
		return nil, err
	}
	return v.(*mtproto.Messages_StickerSet), nil
}

// doFetchAndCacheStickerSet fetches a sticker set from Telegram Bot API, downloads all files
// to DFS synchronously, saves everything to DB, and returns the response.
func (c *StickersCore) doFetchAndCacheStickerSet(shortName string) (*mtproto.Messages_StickerSet, error) {
	startTotal := time.Now()

	botResult, err := c.svcCtx.Dao.BotAPI.GetStickerSet(c.ctx, shortName)
	if err != nil {
		c.Logger.Errorf("doFetchAndCacheStickerSet - BotAPI.GetStickerSet(%s) error: %v", shortName, err)
		return nil, mtproto.ErrStickerIdInvalid
	}

	c.Logger.Infof("doFetchAndCacheStickerSet(%s) - got %d stickers from Bot API in %v",
		shortName, len(botResult.Stickers), time.Since(startTotal))

	// Generate set IDs
	setId := c.svcCtx.Dao.IDGenClient2.NextId(c.ctx)
	setAccessHash := rand.Int63()
	now := time.Now().Unix()

	dataJson, _ := json.Marshal(botResult)

	// Build download inputs for each sticker
	inputs := make([]dao.StickerDownloadInput, 0, len(botResult.Stickers))
	for _, sticker := range botResult.Stickers {
		input := dao.StickerDownloadInput{
			BotFileId:       sticker.FileId,
			BotFileUniqueId: sticker.FileUniqueId,
			MimeType:        stickerMimeType(sticker),
			Attributes:      buildDocumentAttributes(sticker, setId, setAccessHash),
		}
		if sticker.Thumbnail != nil {
			input.ThumbFileId = sticker.Thumbnail.FileId
			input.ThumbWidth = sticker.Thumbnail.Width
			input.ThumbHeight = sticker.Thumbnail.Height
		}
		inputs = append(inputs, input)
	}

	// Download all files and upload to DFS synchronously
	dfsDocs, err := c.svcCtx.Dao.DownloadAndUploadStickerFiles(c.ctx, inputs)
	if err != nil {
		c.Logger.Errorf("doFetchAndCacheStickerSet - DownloadAndUploadStickerFiles(%s) error: %v", shortName, err)
		return nil, mtproto.ErrInternelServerError
	}

	// Build document DOs from DFS results (real DFS-assigned IDs)
	stickerDocDOs := make([]*dataobject.StickerSetDocumentsDO, 0, len(dfsDocs))
	for idx, dfsDoc := range dfsDocs {
		sticker := botResult.Stickers[idx]

		docData, err := dao.SerializeStickerDoc(dfsDoc)
		if err != nil {
			c.Logger.Errorf("doFetchAndCacheStickerSet - serialize dfsDoc error: %v", err)
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

	// Set ThumbDocId to first document for set-level thumbnail (triggers flags bit 8)
	thumbDocId := int64(0)
	if len(dfsDocs) > 0 && dfsDocs[0] != nil {
		thumbDocId = dfsDocs[0].GetId()
	}

	// Compute hash from DFS document IDs
	setHash := computeStickerSetHashFromDocs(dfsDocs)

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
		Hash:         setHash,
		ThumbDocId:   thumbDocId,
		DataJson:     string(dataJson),
		FetchedAt:    now,
	}

	_, rowsAffected, err := c.svcCtx.Dao.StickerSetsDAO.InsertIgnore(c.ctx, setDO)
	if err != nil {
		c.Logger.Errorf("doFetchAndCacheStickerSet - InsertIgnore sticker_sets error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	// Another concurrent request already inserted this set — fall back to cached data
	if rowsAffected == 0 {
		c.Logger.Infof("doFetchAndCacheStickerSet - set %s already cached by another request, falling back", shortName)
		cachedDO, err2 := c.svcCtx.Dao.StickerSetsDAO.SelectByShortName(c.ctx, shortName)
		if err2 != nil || cachedDO == nil {
			c.Logger.Errorf("doFetchAndCacheStickerSet - fallback SelectByShortName(%s) error: %v", shortName, err2)
			return nil, mtproto.ErrInternelServerError
		}
		return c.buildStickerSetFromCache(cachedDO, 0)
	}

	for _, docDO := range stickerDocDOs {
		_, _, err = c.svcCtx.Dao.StickerSetDocumentsDAO.InsertIgnore(c.ctx, docDO)
		if err != nil {
			c.Logger.Errorf("doFetchAndCacheStickerSet - InsertIgnore sticker_set_documents error: %v", err)
		}
	}

	packs := buildStickerPacks2(stickerDocDOs)
	stickerSetPB := makeStickerSetFromDO(setDO)

	c.Logger.Infof("doFetchAndCacheStickerSet(%s) - DONE: %d docs, total=%v",
		shortName, len(dfsDocs), time.Since(startTotal))

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
	attrs := make([]*mtproto.DocumentAttribute, 0, 4)

	attrs = append(attrs, mtproto.MakeTLDocumentAttributeSticker(&mtproto.DocumentAttribute{
		Alt: s.Emoji,
		Stickerset: mtproto.MakeTLInputStickerSetID(&mtproto.InputStickerSet{
			Id:         setId,
			AccessHash: setAccessHash,
		}).To_InputStickerSet(),
	}).To_DocumentAttribute())

	if s.IsVideo {
		attrs = append(attrs, mtproto.MakeTLDocumentAttributeVideo(&mtproto.DocumentAttribute{
			W:        s.Width,
			H:        s.Height,
			Duration: 0,
		}).To_DocumentAttribute())
	} else {
		attrs = append(attrs, mtproto.MakeTLDocumentAttributeImageSize(&mtproto.DocumentAttribute{
			W: s.Width,
			H: s.Height,
		}).To_DocumentAttribute())
	}

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

// systemBuiltInPredicates maps system built-in sticker set predicates to their shortNames.
var systemBuiltInPredicates = map[string]string{
	mtproto.Predicate_inputStickerSetAnimatedEmoji:             "AnimatedEmojies",
	mtproto.Predicate_inputStickerSetAnimatedEmojiAnimations:   "EmojiAnimations",
	mtproto.Predicate_inputStickerSetEmojiGenericAnimations:    "EmojiGenericAnimations",
	mtproto.Predicate_inputStickerSetEmojiDefaultStatuses:      "StatusPack",
	mtproto.Predicate_inputStickerSetEmojiDefaultTopicIcons:    "Topics",
}

func isSystemBuiltInPredicate(predicate string) bool {
	_, ok := systemBuiltInPredicates[predicate]
	return ok
}

// computeStickerSetHash computes the Telegram-standard hash for a StickerSet from its document DOs.
func computeStickerSetHash(docDOs []dataobject.StickerSetDocumentsDO) int32 {
	var acc uint64
	for _, d := range docDOs {
		telegramCombineInt64Hash(&acc, uint64(d.DocumentId))
	}
	return int32(acc)
}

// computeStickerSetHashFromDocs computes the hash from DFS Document objects (used during initial fetch).
func computeStickerSetHashFromDocs(docs []*mtproto.Document) int32 {
	var acc uint64
	for _, doc := range docs {
		if doc != nil {
			telegramCombineInt64Hash(&acc, uint64(doc.GetId()))
		}
	}
	return int32(acc)
}

// makeEmptyStickerSet returns a valid but empty Messages_StickerSet for system built-in sets
// that cannot be fetched from Bot API. This prevents the client from receiving STICKER_ID_INVALID.
func (c *StickersCore) makeEmptyStickerSet(shortName string) *mtproto.Messages_StickerSet {
	return mtproto.MakeTLMessagesStickerSet(&mtproto.Messages_StickerSet{
		Set: mtproto.MakeTLStickerSet(&mtproto.StickerSet{
			Id:        0,
			Title:     shortName,
			ShortName: shortName,
			Count:     0,
			Hash:      0,
		}).To_StickerSet(),
		Packs:     []*mtproto.StickerPack{},
		Keywords:  []*mtproto.StickerKeyword{},
		Documents: []*mtproto.Document{},
	}).To_Messages_StickerSet()
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
