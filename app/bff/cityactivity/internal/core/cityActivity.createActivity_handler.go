package core

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/cityactivity/internal/dao"
)

func (c *CityActivityCore) CityActivityCreateActivity(in *mtproto.TLCityActivityCreateActivity) (*mtproto.CityActivity, error) {
	if c.MD == nil {
		return nil, mtproto.ErrInternelServerError
	}
	city := in.GetCity()
	if city == "" && c.MD.ClientAddr != "" {
		city = c.svcCtx.Dao.GetCityByIp(c.MD.ClientAddr)
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
	}

	id, err := c.svcCtx.Dao.CreateActivity(c.ctx, a)
	if err != nil {
		c.Logger.Errorf("cityActivity.createActivity - error: %v", err)
		return nil, err
	}
	a.Id = id

	return activityToProto(a, false), nil
}
