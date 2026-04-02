package core

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/cityactivity/internal/dao"
	chatpb "github.com/teamgram/teamgram-server/app/service/biz/chat/chat"
	media "github.com/teamgram/teamgram-server/app/service/media/media"
)

func (c *CityActivityCore) CityActivityCreateActivity(in *mtproto.TLCityActivityCreateActivity) (*mtproto.CityActivity, error) {
	if c.MD == nil {
		return nil, mtproto.ErrInternelServerError
	}
	city := in.GetCity()
	if city == "" && c.MD.ClientAddr != "" {
		city = c.svcCtx.Dao.GetCityByIp(c.MD.ClientAddr)
	}

	isGlobal := int32(0)
	if in.GetIsGlobal() != nil && mtproto.FromBool(in.GetIsGlobal()) {
		isGlobal = 1
	}

	a := &dao.Activity{
		UserId:          c.MD.UserId,
		Title:           in.GetTitle(),
		Description:     in.GetDescription(),
		PhotoId:         in.GetPhotoId(),
		City:            city,
		StartTime:       in.GetStartTime(),
		EndTime:         in.GetEndTime(),
		MaxParticipants: in.GetMaxParticipants(),
		IsGlobal:        isGlobal,
	}

	id, err := c.svcCtx.Dao.CreateActivity(c.ctx, a)
	if err != nil {
		c.Logger.Errorf("cityActivity.createActivity - error: %v", err)
		return nil, err
	}
	a.Id = id

	// Save activity media (photo_ids)
	photoIds := in.GetPhotoIds()
	if len(photoIds) > 5 {
		photoIds = photoIds[:5]
	}
	if len(photoIds) > 0 {
		if err2 := c.svcCtx.Dao.SaveActivityMedia(c.ctx, id, photoIds); err2 != nil {
			c.Logger.Errorf("cityActivity.createActivity - save media error: %v", err2)
		}
	}

	// Auto-create group chat
	chat, err := c.svcCtx.Dao.ChatCreateChat2(c.ctx, &chatpb.TLChatCreateChat2{
		CreatorId:  c.MD.UserId,
		UserIdList: []int64{},
		Title:      in.GetTitle(),
	})
	if err != nil {
		c.Logger.Errorf("cityActivity.createActivity - create chat error: %v", err)
	} else if chat != nil && chat.Chat != nil {
		a.ChatId = chat.Chat.Id
		if err2 := c.svcCtx.Dao.UpdateActivityChatId(c.ctx, id, chat.Chat.Id); err2 != nil {
			c.Logger.Errorf("cityActivity.createActivity - update chat_id error: %v", err2)
		}
	}

	// Creator auto-join activity
	if err2 := c.svcCtx.Dao.JoinActivity(c.ctx, id, c.MD.UserId, city); err2 != nil {
		c.Logger.Errorf("cityActivity.createActivity - creator join error: %v", err2)
	}
	a.ParticipantCount = 1

	// Resolve photos for response
	var photos []*mtproto.Photo
	for _, pid := range photoIds {
		photo, err3 := c.svcCtx.Dao.MediaGetPhoto(c.ctx, &media.TLMediaGetPhoto{PhotoId: pid})
		if err3 == nil && photo != nil {
			photos = append(photos, photo)
		}
	}

	result := activityToProto(a, true)
	result.Photos = photos
	return result, nil
}
