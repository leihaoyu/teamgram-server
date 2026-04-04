package core

import (
	"github.com/teamgram/proto/mtproto"
	userpb "github.com/teamgram/teamgram-server/app/service/biz/user/user"
	media "github.com/teamgram/teamgram-server/app/service/media/media"
)

func (c *CityActivityCore) CityActivityGetMyActivities(in *mtproto.TLCityActivityGetMyActivities) (*mtproto.CityActivity_Activities, error) {
	var userId int64
	if c.MD != nil {
		userId = c.MD.UserId
	}
	if userId == 0 {
		return nil, mtproto.ErrAuthKeyPermEmpty
	}

	offset := in.GetOffset()
	limit := in.GetLimit()
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	activities, count, err := c.svcCtx.Dao.GetMyActivities(c.ctx, userId, offset, limit)
	if err != nil {
		c.Logger.Errorf("cityActivity.getMyActivities - error: %v", err)
		return nil, err
	}

	// Batch get first photo for each activity
	activityIds := make([]int64, 0, len(activities))
	for _, a := range activities {
		activityIds = append(activityIds, a.Id)
	}
	firstPhotoIds, _ := c.svcCtx.Dao.GetActivitiesFirstPhotoIds(c.ctx, activityIds)

	// Batch resolve creator names
	creatorIds := make([]int64, 0, len(activities))
	for _, a := range activities {
		creatorIds = append(creatorIds, a.UserId)
	}
	creatorNames := make(map[int64]string)
	if len(creatorIds) > 0 {
		if userData, err2 := c.svcCtx.Dao.UserClient.UserGetUserDataListByIdList(c.ctx, &userpb.TLUserGetUserDataListByIdList{
			UserIdList: creatorIds,
		}); err2 == nil {
			for _, ud := range userData.GetDatas() {
				name := ud.GetFirstName()
				if ln := ud.GetLastName(); ln != "" {
					name += " " + ln
				}
				creatorNames[ud.GetId()] = name
			}
		}
	}
	for _, a := range activities {
		if name, ok := creatorNames[a.UserId]; ok {
			a.CreatorName = name
		}
	}

	// Resolve photos via MediaClient
	photoCache := make(map[int64]*mtproto.Photo)
	for _, pid := range firstPhotoIds {
		if _, ok := photoCache[pid]; !ok {
			photo, err2 := c.svcCtx.Dao.MediaGetPhoto(c.ctx, &media.TLMediaGetPhoto{PhotoId: pid})
			if err2 == nil && photo != nil {
				photoCache[pid] = photo
			}
		}
	}

	result := mtproto.MakeTLCityActivityActivities(&mtproto.CityActivity_Activities{
		Count:      count,
		Activities: make([]*mtproto.CityActivity, 0, len(activities)),
	})

	for _, a := range activities {
		isJoined := c.svcCtx.Dao.IsUserJoined(c.ctx, a.Id, userId)
		proto := activityToProto(a, isJoined)
		if pid, ok := firstPhotoIds[a.Id]; ok {
			if photo, ok2 := photoCache[pid]; ok2 {
				proto.Photos = []*mtproto.Photo{photo}
			}
		}
		result.Data2.Activities = append(result.Data2.Activities, proto)
	}

	return result.To_CityActivity_Activities(), nil
}
