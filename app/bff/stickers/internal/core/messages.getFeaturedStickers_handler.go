package core

import (
	"context"

	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dao"
	"github.com/zeromicro/go-zero/core/logx"
)

// MessagesGetFeaturedStickers returns popular/featured sticker sets.
func (c *StickersCore) MessagesGetFeaturedStickers(in *mtproto.TLMessagesGetFeaturedStickers) (*mtproto.Messages_FeaturedStickers, error) {
	return c.getFeaturedStickersByType(in.Hash, 0)
}

// MessagesGetFeaturedEmojiStickers returns popular/featured emoji sticker sets.
func (c *StickersCore) MessagesGetFeaturedEmojiStickers(in *mtproto.TLMessagesGetFeaturedEmojiStickers) (*mtproto.Messages_FeaturedStickers, error) {
	return c.getFeaturedStickersByType(in.Hash, 2)
}

func (c *StickersCore) getFeaturedStickersByType(clientHash int64, setType int32) (*mtproto.Messages_FeaturedStickers, error) {
	const featuredLimit int32 = 20

	// 1. Get popular set_ids from install data
	var (
		popularSetIds []int64
		err           error
	)
	if setType == 2 {
		popularSetIds, err = c.svcCtx.Dao.UserInstalledStickerSetsDAO.SelectPopularEmojiSetIds(c.ctx, featuredLimit)
	} else {
		popularSetIds, err = c.svcCtx.Dao.UserInstalledStickerSetsDAO.SelectPopularSetIds(c.ctx, featuredLimit)
	}
	if err != nil {
		c.Logger.Errorf("getFeaturedStickersByType(%d) - SelectPopularSetIds error: %v", setType, err)
		return nil, mtproto.ErrInternelServerError
	}

	// 2. Cold-start fallback: supplement with configured set names
	if len(popularSetIds) < int(featuredLimit) {
		var configuredNames []string
		if setType == 2 {
			configuredNames = c.svcCtx.Config.FeaturedEmojiStickerSets
		} else {
			configuredNames = c.svcCtx.Config.FeaturedStickerSets
		}
		popularSetIds = c.supplementWithConfiguredSets(popularSetIds, int(featuredLimit), configuredNames)
	}

	// 3. Exclude user's installed sets
	installedSetIds := c.getInstalledSetIdMap(setType)
	filteredIds := make([]int64, 0, len(popularSetIds))
	for _, id := range popularSetIds {
		if !installedSetIds[id] {
			filteredIds = append(filteredIds, id)
		}
	}

	if len(filteredIds) == 0 {
		return mtproto.MakeTLMessagesFeaturedStickers(&mtproto.Messages_FeaturedStickers{
			Count:  0,
			Hash:   0,
			Sets:   []*mtproto.StickerSetCovered{},
			Unread: []int64{},
		}).To_Messages_FeaturedStickers(), nil
	}

	// 4. Build StickerSetCovered for each set
	sets, err := c.buildStickerSetsCovered(filteredIds)
	if err != nil {
		c.Logger.Errorf("getFeaturedStickersByType(%d) - buildStickerSetsCovered error: %v", setType, err)
		return nil, mtproto.ErrInternelServerError
	}

	// 5. Compute hash over set IDs
	var hashAcc uint64
	for _, s := range sets {
		if s.Set != nil {
			telegramCombineInt64Hash(&hashAcc, uint64(s.Set.Id))
		}
	}
	hash := int64(hashAcc)

	// 6. Check NotModified
	if clientHash != 0 && clientHash == hash {
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
func (c *StickersCore) supplementWithConfiguredSets(existingIds []int64, maxLen int, configuredNames []string) []int64 {
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
			// Not cached yet — skip it to avoid blocking this request.
			// Trigger async background download so it's ready for the next request.
			c.Logger.Infof("supplementWithConfiguredSets - set %s not cached, triggering async fetch", name)
			svcCtx := c.svcCtx
			setName := name
			go func() {
				_, _ = svcCtx.Dao.StickerSetFetch.Do(setName, func() error {
					bgCore := New(context.Background(), svcCtx)
					_, err := bgCore.fetchAndCacheStickerSet(setName)
					if err != nil {
						logx.Errorf("async fetchAndCacheStickerSet(%s) error: %v", setName, err)
					} else {
						logx.Infof("async fetchAndCacheStickerSet(%s) done", setName)
					}
					return err
				})
			}()
			continue
		}

		if !existingSet[setDO.SetId] {
			result = append(result, setDO.SetId)
			existingSet[setDO.SetId] = true
		}
	}

	return result
}

// getInstalledSetIdMap returns the current user's installed set_ids as a map for fast lookup.
func (c *StickersCore) getInstalledSetIdMap(setType int32) map[int64]bool {
	installedRows, err := c.svcCtx.Dao.UserInstalledStickerSetsDAO.SelectByUserAndType(c.ctx, c.MD.UserId, setType)
	if err != nil {
		c.Logger.Errorf("getInstalledSetIdMap - error: %v", err)
		return nil
	}
	m := make(map[int64]bool, len(installedRows))
	for _, r := range installedRows {
		m[r.SetId] = true
	}
	return m
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
