package service

import (
	"context"

	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/core"
)

// Embed UnimplementedRPCStickersServer so all 30 methods have defaults.
var _ mtproto.RPCStickersServer = (*Service)(nil)

// MessagesGetStickerSet implements the real handler.
func (s *Service) MessagesGetStickerSet(ctx context.Context, request *mtproto.TLMessagesGetStickerSet) (*mtproto.Messages_StickerSet, error) {
	c := core.New(ctx, s.svcCtx)
	c.Logger.Debugf("messages.getStickerSet - metadata: %s, request: %s", c.MD, request)

	r, err := c.MessagesGetStickerSet(request)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// --- All other RPCStickersServer methods return "not implemented" ---

func (s *Service) MessagesGetStickers(ctx context.Context, request *mtproto.TLMessagesGetStickers) (*mtproto.Messages_Stickers, error) {
	return mtproto.MakeTLMessagesStickers(&mtproto.Messages_Stickers{
		Hash:     0,
		Stickers: []*mtproto.Document{},
	}).To_Messages_Stickers(), nil
}

func (s *Service) MessagesGetAllStickers(ctx context.Context, request *mtproto.TLMessagesGetAllStickers) (*mtproto.Messages_AllStickers, error) {
	return mtproto.MakeTLMessagesAllStickers(&mtproto.Messages_AllStickers{
		Hash: 0,
		Sets: []*mtproto.StickerSet{},
	}).To_Messages_AllStickers(), nil
}

func (s *Service) MessagesInstallStickerSet(ctx context.Context, request *mtproto.TLMessagesInstallStickerSet) (*mtproto.Messages_StickerSetInstallResult, error) {
	return mtproto.MakeTLMessagesStickerSetInstallResultSuccess(&mtproto.Messages_StickerSetInstallResult{}).To_Messages_StickerSetInstallResult(), nil
}

func (s *Service) MessagesUninstallStickerSet(ctx context.Context, request *mtproto.TLMessagesUninstallStickerSet) (*mtproto.Bool, error) {
	return mtproto.BoolTrue, nil
}

func (s *Service) MessagesReorderStickerSets(ctx context.Context, request *mtproto.TLMessagesReorderStickerSets) (*mtproto.Bool, error) {
	return mtproto.BoolTrue, nil
}

func (s *Service) MessagesGetFeaturedStickers(ctx context.Context, request *mtproto.TLMessagesGetFeaturedStickers) (*mtproto.Messages_FeaturedStickers, error) {
	return mtproto.MakeTLMessagesFeaturedStickers(&mtproto.Messages_FeaturedStickers{
		Count:  0,
		Hash:   0,
		Sets:   []*mtproto.StickerSetCovered{},
		Unread: []int64{},
	}).To_Messages_FeaturedStickers(), nil
}

func (s *Service) MessagesReadFeaturedStickers(ctx context.Context, request *mtproto.TLMessagesReadFeaturedStickers) (*mtproto.Bool, error) {
	return mtproto.BoolTrue, nil
}

func (s *Service) MessagesGetRecentStickers(ctx context.Context, request *mtproto.TLMessagesGetRecentStickers) (*mtproto.Messages_RecentStickers, error) {
	c := core.New(ctx, s.svcCtx)
	c.Logger.Debugf("messages.getRecentStickers - metadata: %s, request: %s", c.MD, request)
	return c.MessagesGetRecentStickers(request)
}

func (s *Service) MessagesSaveRecentSticker(ctx context.Context, request *mtproto.TLMessagesSaveRecentSticker) (*mtproto.Bool, error) {
	c := core.New(ctx, s.svcCtx)
	c.Logger.Debugf("messages.saveRecentSticker - metadata: %s, request: %s", c.MD, request)
	return c.MessagesSaveRecentSticker(request)
}

func (s *Service) MessagesClearRecentStickers(ctx context.Context, request *mtproto.TLMessagesClearRecentStickers) (*mtproto.Bool, error) {
	c := core.New(ctx, s.svcCtx)
	c.Logger.Debugf("messages.clearRecentStickers - metadata: %s, request: %s", c.MD, request)
	return c.MessagesClearRecentStickers(request)
}

func (s *Service) MessagesGetArchivedStickers(ctx context.Context, request *mtproto.TLMessagesGetArchivedStickers) (*mtproto.Messages_ArchivedStickers, error) {
	return mtproto.MakeTLMessagesArchivedStickers(&mtproto.Messages_ArchivedStickers{
		Count: 0,
		Sets:  []*mtproto.StickerSetCovered{},
	}).To_Messages_ArchivedStickers(), nil
}

func (s *Service) MessagesGetMaskStickers(ctx context.Context, request *mtproto.TLMessagesGetMaskStickers) (*mtproto.Messages_AllStickers, error) {
	return mtproto.MakeTLMessagesAllStickers(&mtproto.Messages_AllStickers{
		Hash: 0,
		Sets: []*mtproto.StickerSet{},
	}).To_Messages_AllStickers(), nil
}

func (s *Service) MessagesGetAttachedStickers(ctx context.Context, request *mtproto.TLMessagesGetAttachedStickers) (*mtproto.Vector_StickerSetCovered, error) {
	return &mtproto.Vector_StickerSetCovered{
		Datas: []*mtproto.StickerSetCovered{},
	}, nil
}

func (s *Service) MessagesGetFavedStickers(ctx context.Context, request *mtproto.TLMessagesGetFavedStickers) (*mtproto.Messages_FavedStickers, error) {
	c := core.New(ctx, s.svcCtx)
	c.Logger.Debugf("messages.getFavedStickers - metadata: %s, request: %s", c.MD, request)
	return c.MessagesGetFavedStickers(request)
}

func (s *Service) MessagesFaveSticker(ctx context.Context, request *mtproto.TLMessagesFaveSticker) (*mtproto.Bool, error) {
	c := core.New(ctx, s.svcCtx)
	c.Logger.Debugf("messages.faveSticker - metadata: %s, request: %s", c.MD, request)
	return c.MessagesFaveSticker(request)
}

func (s *Service) MessagesSearchStickerSets(ctx context.Context, request *mtproto.TLMessagesSearchStickerSets) (*mtproto.Messages_FoundStickerSets, error) {
	return mtproto.MakeTLMessagesFoundStickerSets(&mtproto.Messages_FoundStickerSets{
		Hash: 0,
		Sets: []*mtproto.StickerSetCovered{},
	}).To_Messages_FoundStickerSets(), nil
}

func (s *Service) MessagesToggleStickerSets(ctx context.Context, request *mtproto.TLMessagesToggleStickerSets) (*mtproto.Bool, error) {
	return mtproto.BoolTrue, nil
}

func (s *Service) MessagesGetOldFeaturedStickers(ctx context.Context, request *mtproto.TLMessagesGetOldFeaturedStickers) (*mtproto.Messages_FeaturedStickers, error) {
	return mtproto.MakeTLMessagesFeaturedStickers(&mtproto.Messages_FeaturedStickers{
		Count:  0,
		Hash:   0,
		Sets:   []*mtproto.StickerSetCovered{},
		Unread: []int64{},
	}).To_Messages_FeaturedStickers(), nil
}

func (s *Service) MessagesSearchEmojiStickerSets(ctx context.Context, request *mtproto.TLMessagesSearchEmojiStickerSets) (*mtproto.Messages_FoundStickerSets, error) {
	return mtproto.MakeTLMessagesFoundStickerSets(&mtproto.Messages_FoundStickerSets{
		Hash: 0,
		Sets: []*mtproto.StickerSetCovered{},
	}).To_Messages_FoundStickerSets(), nil
}

func (s *Service) StickersCreateStickerSet(ctx context.Context, request *mtproto.TLStickersCreateStickerSet) (*mtproto.Messages_StickerSet, error) {
	return nil, mtproto.ErrMethodNotImpl
}

func (s *Service) StickersRemoveStickerFromSet(ctx context.Context, request *mtproto.TLStickersRemoveStickerFromSet) (*mtproto.Messages_StickerSet, error) {
	return nil, mtproto.ErrMethodNotImpl
}

func (s *Service) StickersChangeStickerPosition(ctx context.Context, request *mtproto.TLStickersChangeStickerPosition) (*mtproto.Messages_StickerSet, error) {
	return nil, mtproto.ErrMethodNotImpl
}

func (s *Service) StickersAddStickerToSet(ctx context.Context, request *mtproto.TLStickersAddStickerToSet) (*mtproto.Messages_StickerSet, error) {
	return nil, mtproto.ErrMethodNotImpl
}

func (s *Service) StickersSetStickerSetThumb(ctx context.Context, request *mtproto.TLStickersSetStickerSetThumb) (*mtproto.Messages_StickerSet, error) {
	return nil, mtproto.ErrMethodNotImpl
}

func (s *Service) StickersCheckShortName(ctx context.Context, request *mtproto.TLStickersCheckShortName) (*mtproto.Bool, error) {
	return mtproto.BoolFalse, nil
}

func (s *Service) StickersSuggestShortName(ctx context.Context, request *mtproto.TLStickersSuggestShortName) (*mtproto.Stickers_SuggestedShortName, error) {
	return nil, mtproto.ErrMethodNotImpl
}

func (s *Service) StickersChangeSticker(ctx context.Context, request *mtproto.TLStickersChangeSticker) (*mtproto.Messages_StickerSet, error) {
	return nil, mtproto.ErrMethodNotImpl
}

func (s *Service) StickersRenameStickerSet(ctx context.Context, request *mtproto.TLStickersRenameStickerSet) (*mtproto.Messages_StickerSet, error) {
	return nil, mtproto.ErrMethodNotImpl
}

func (s *Service) StickersDeleteStickerSet(ctx context.Context, request *mtproto.TLStickersDeleteStickerSet) (*mtproto.Bool, error) {
	return mtproto.BoolTrue, nil
}
