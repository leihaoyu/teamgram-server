package core

import (
	"github.com/teamgram/proto/mtproto"
	userpb "github.com/teamgram/teamgram-server/app/service/biz/user/user"
	media "github.com/teamgram/teamgram-server/app/service/media/media"
)

func (c *CityActivityCore) CityActivityGetActivity(in *mtproto.TLCityActivityGetActivity) (*mtproto.CityActivity, error) {
	activity, err := c.svcCtx.Dao.GetActivityById(c.ctx, in.GetId())
	if err != nil {
		c.Logger.Errorf("cityActivity.getActivity - error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}
	if activity == nil {
		return nil, mtproto.ErrInternelServerError
	}

	var userId int64
	if c.MD != nil {
		userId = c.MD.UserId
	}
	isJoined := false
	if userId > 0 {
		isJoined = c.svcCtx.Dao.IsUserJoined(c.ctx, activity.Id, userId)
	}

	// Resolve creator name
	if userData, err2 := c.svcCtx.Dao.UserClient.UserGetUserDataListByIdList(c.ctx, &userpb.TLUserGetUserDataListByIdList{
		UserIdList: []int64{activity.UserId},
	}); err2 == nil {
		for _, ud := range userData.GetDatas() {
			name := ud.GetFirstName()
			if ln := ud.GetLastName(); ln != "" {
				name += " " + ln
			}
			activity.CreatorName = name
		}
	}

	result := activityToProto(activity, isJoined)

	// Resolve all photos for detail view
	photoIds, err2 := c.svcCtx.Dao.GetActivityPhotoIds(c.ctx, activity.Id)
	if err2 == nil && len(photoIds) > 0 {
		var photos []*mtproto.Photo
		for _, pid := range photoIds {
			photo, err3 := c.svcCtx.Dao.MediaGetPhoto(c.ctx, &media.TLMediaGetPhoto{PhotoId: pid})
			if err3 != nil {
				c.Logger.Errorf("cityActivity.getActivity - MediaGetPhoto(%d) error: %v", pid, err3)
			} else if photo != nil {
				photos = append(photos, photo)
			}
		}
		result.Photos = photos
	} else if err2 != nil {
		c.Logger.Errorf("cityActivity.getActivity - GetActivityPhotoIds error: %v", err2)
	}

	return result, nil
}
