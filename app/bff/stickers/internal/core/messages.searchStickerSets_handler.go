package core

import (
	"github.com/teamgram/proto/mtproto"
)

// MessagesSearchStickerSets searches sticker sets by title/short_name with Bot API fallback.
func (c *StickersCore) MessagesSearchStickerSets(in *mtproto.TLMessagesSearchStickerSets) (*mtproto.Messages_FoundStickerSets, error) {
	q := in.Q

	if q == "" {
		return mtproto.MakeTLMessagesFoundStickerSets(&mtproto.Messages_FoundStickerSets{
			Hash: 0,
			Sets: []*mtproto.StickerSetCovered{},
		}).To_Messages_FoundStickerSets(), nil
	}

	// 1. Search local DB by title/short_name LIKE
	searchResults, err := c.svcCtx.Dao.StickerSetsDAO.SearchByQuery(c.ctx, q, 20)
	if err != nil {
		c.Logger.Errorf("messages.searchStickerSets - SearchByQuery(%s) error: %v", q, err)
		return nil, mtproto.ErrInternelServerError
	}

	// 2. If no local results, try Bot API exact name match as fallback (with singleflight dedup)
	if len(searchResults) == 0 {
		_, sfErr := c.svcCtx.Dao.StickerSetFetch.Do(q, func() error {
			_, fetchErr := c.fetchAndCacheStickerSet(q)
			return fetchErr
		})
		if sfErr == nil {
			// Re-query after caching
			searchResults, err = c.svcCtx.Dao.StickerSetsDAO.SearchByQuery(c.ctx, q, 20)
			if err != nil {
				c.Logger.Errorf("messages.searchStickerSets - re-SearchByQuery(%s) error: %v", q, err)
				return nil, mtproto.ErrInternelServerError
			}
		}
	}

	if len(searchResults) == 0 {
		return mtproto.MakeTLMessagesFoundStickerSets(&mtproto.Messages_FoundStickerSets{
			Hash: 0,
			Sets: []*mtproto.StickerSetCovered{},
		}).To_Messages_FoundStickerSets(), nil
	}

	// 3. Exclude user's installed sets to avoid stableId collision on iOS
	installedSetIds := c.getInstalledSetIdMap()

	setIds := make([]int64, 0, len(searchResults))
	for i := range searchResults {
		if !installedSetIds[searchResults[i].SetId] {
			setIds = append(setIds, searchResults[i].SetId)
		}
	}

	if len(setIds) == 0 {
		return mtproto.MakeTLMessagesFoundStickerSets(&mtproto.Messages_FoundStickerSets{
			Hash: 0,
			Sets: []*mtproto.StickerSetCovered{},
		}).To_Messages_FoundStickerSets(), nil
	}

	// 4. Build StickerSetCovered for each result
	sets, err := c.buildStickerSetsCovered(setIds)
	if err != nil {
		c.Logger.Errorf("messages.searchStickerSets - buildStickerSetsCovered error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	// 5. Compute hash
	var hashAcc uint64
	for _, s := range sets {
		if s.Set != nil {
			telegramCombineInt64Hash(&hashAcc, uint64(s.Set.Id))
		}
	}
	hash := int64(hashAcc)

	// 6. Check NotModified
	if in.Hash != 0 && in.Hash == hash {
		return mtproto.MakeTLMessagesFoundStickerSetsNotModified(nil).To_Messages_FoundStickerSets(), nil
	}

	return mtproto.MakeTLMessagesFoundStickerSets(&mtproto.Messages_FoundStickerSets{
		Hash: hash,
		Sets: sets,
	}).To_Messages_FoundStickerSets(), nil
}
