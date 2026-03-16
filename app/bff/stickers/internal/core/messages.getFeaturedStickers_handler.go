package core

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dao"
)

// MessagesGetFeaturedStickers returns popular/featured sticker sets.
func (c *StickersCore) MessagesGetFeaturedStickers(in *mtproto.TLMessagesGetFeaturedStickers) (*mtproto.Messages_FeaturedStickers, error) {
	const featuredLimit int32 = 20

	// 1. Get popular set_ids from install data
	popularSetIds, err := c.svcCtx.Dao.UserInstalledStickerSetsDAO.SelectPopularSetIds(c.ctx, featuredLimit)
	if err != nil {
		c.Logger.Errorf("messages.getFeaturedStickers - SelectPopularSetIds error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	// 2. Cold-start fallback: supplement with configured set names
	if len(popularSetIds) < int(featuredLimit) {
		popularSetIds = c.supplementWithConfiguredSets(popularSetIds, int(featuredLimit))
	}

	if len(popularSetIds) == 0 {
		return mtproto.MakeTLMessagesFeaturedStickers(&mtproto.Messages_FeaturedStickers{
			Count:  0,
			Hash:   0,
			Sets:   []*mtproto.StickerSetCovered{},
			Unread: []int64{},
		}).To_Messages_FeaturedStickers(), nil
	}

	// 3. Build StickerSetCovered for each set
	sets, err := c.buildStickerSetsCovered(popularSetIds)
	if err != nil {
		c.Logger.Errorf("messages.getFeaturedStickers - buildStickerSetsCovered error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	// 4. Compute hash over set IDs
	var hashAcc uint64
	for _, s := range sets {
		if s.Set != nil {
			telegramCombineInt64Hash(&hashAcc, uint64(s.Set.Id))
		}
	}
	hash := int64(hashAcc)

	// 5. Check NotModified
	if in.Hash != 0 && in.Hash == hash {
		return mtproto.MakeTLMessagesFeaturedStickersNotModified(nil).To_Messages_FeaturedStickers(), nil
	}

	return mtproto.MakeTLMessagesFeaturedStickers(&mtproto.Messages_FeaturedStickers{
		Count:  int32(len(sets)),
		Hash:   hash,
		Sets:   sets,
		Unread: []int64{},
	}).To_Messages_FeaturedStickers(), nil
}

// supplementWithConfiguredSets adds configured featured set short_names not already in the list.
func (c *StickersCore) supplementWithConfiguredSets(existingIds []int64, maxLen int) []int64 {
	configuredNames := c.svcCtx.Config.FeaturedStickerSets
	if len(configuredNames) == 0 {
		return existingIds
	}

	existingSet := make(map[int64]bool, len(existingIds))
	for _, id := range existingIds {
		existingSet[id] = true
	}

	result := make([]int64, len(existingIds))
	copy(result, existingIds)

	for _, name := range configuredNames {
		if len(result) >= maxLen {
			break
		}

		setDO, err := c.svcCtx.Dao.StickerSetsDAO.SelectByShortName(c.ctx, name)
		if err != nil {
			c.Logger.Errorf("supplementWithConfiguredSets - SelectByShortName(%s) error: %v", name, err)
			continue
		}

		if setDO == nil {
			// Not cached yet — fetch from Bot API
			_, err = c.fetchAndCacheStickerSet(name)
			if err != nil {
				c.Logger.Errorf("supplementWithConfiguredSets - fetchAndCacheStickerSet(%s) error: %v", name, err)
				continue
			}
			setDO, err = c.svcCtx.Dao.StickerSetsDAO.SelectByShortName(c.ctx, name)
			if err != nil || setDO == nil {
				continue
			}
		}

		if !existingSet[setDO.SetId] {
			result = append(result, setDO.SetId)
			existingSet[setDO.SetId] = true
		}
	}

	return result
}

// buildStickerSetsCovered builds StickerSetCovered objects from a list of set_ids.
func (c *StickersCore) buildStickerSetsCovered(setIds []int64) ([]*mtproto.StickerSetCovered, error) {
	setDOs, err := c.svcCtx.Dao.StickerSetsDAO.SelectBySetIds(c.ctx, setIds)
	if err != nil {
		return nil, err
	}

	setDOMap := make(map[int64]*dataobject.StickerSetsDO, len(setDOs))
	for i := range setDOs {
		setDOMap[setDOs[i].SetId] = &setDOs[i]
	}

	result := make([]*mtproto.StickerSetCovered, 0, len(setIds))
	for _, setId := range setIds {
		setDO, ok := setDOMap[setId]
		if !ok {
			continue
		}

		stickerSet := makeStickerSetFromDO(setDO)

		// Get cover document (first document in set)
		var coverDoc *mtproto.Document
		coverDO, err2 := c.svcCtx.Dao.StickerSetDocumentsDAO.SelectFirstBySetId(c.ctx, setId)
		if err2 != nil {
			c.Logger.Errorf("buildStickerSetsCovered - SelectFirstBySetId(%d) error: %v", setId, err2)
		} else if coverDO != nil {
			coverDoc, _ = dao.DeserializeStickerDoc(coverDO.DocumentData)
		}

		covered := mtproto.MakeTLStickerSetCovered(&mtproto.StickerSetCovered{
			Set:   stickerSet,
			Cover: coverDoc,
		}).To_StickerSetCovered()

		result = append(result, covered)
	}

	return result, nil
}
