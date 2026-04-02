package core

import (
	"github.com/teamgram/proto/mtproto"
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

	result := activityToProto(activity, isJoined)

	// Resolve all photos for detail view
	photoIds, err2 := c.svcCtx.Dao.GetActivityPhotoIds(c.ctx, activity.Id)
	if err2 == nil && len(photoIds) > 0 {
		var photos []*mtproto.Photo
		for _, pid := range photoIds {
			photo, err3 := c.svcCtx.Dao.MediaGetPhoto(c.ctx, &media.TLMediaGetPhoto{PhotoId: pid})
			if err3 == nil && photo != nil {
				photos = append(photos, photo)
			}
		}
		result.Photos = photos
	}

	return result, nil
}
