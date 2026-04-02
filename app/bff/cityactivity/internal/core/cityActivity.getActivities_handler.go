package core

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/cityactivity/internal/dao"
)

func (c *CityActivityCore) CityActivityGetActivities(in *mtproto.TLCityActivityGetActivities) (*mtproto.CityActivity_Activities, error) {
	city := in.GetCity()

	// If client didn't specify a city, detect from IP
	if city == "" && c.MD != nil && c.MD.ClientAddr != "" {
		city = c.svcCtx.Dao.GetCityByIp(c.MD.ClientAddr)
		c.Logger.Infof("cityActivity.getActivities - detected city: %q from IP: %s", city, c.MD.ClientAddr)
	}

	offset := in.GetOffset()
	limit := in.GetLimit()
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	activities, count, err := c.svcCtx.Dao.GetActivitiesByCity(c.ctx, city, offset, limit)
	if err != nil {
		c.Logger.Errorf("cityActivity.getActivities - error: %v", err)
		return nil, err
	}

	var userId int64
	if c.MD != nil {
		userId = c.MD.UserId
	}

	result := mtproto.MakeTLCityActivityActivities(&mtproto.CityActivity_Activities{
		Count:      count,
		Activities: make([]*mtproto.CityActivity, 0, len(activities)),
	})

	for _, a := range activities {
		isJoined := false
		if userId > 0 {
			isJoined = c.svcCtx.Dao.IsUserJoined(c.ctx, a.Id, userId)
		}
		result.Data2.Activities = append(result.Data2.Activities, activityToProto(a, isJoined))
	}

	return result.To_CityActivity_Activities(), nil
}

func activityToProto(a *dao.Activity, isJoined bool) *mtproto.CityActivity {
	return mtproto.MakeTLCityActivity(&mtproto.CityActivity{
		Id:               a.Id,
		UserId:           a.UserId,
		Title:            a.Title,
		Description:      a.Description,
		PhotoId:          a.PhotoId,
		City:             a.City,
		StartTime:        a.StartTime,
		EndTime:          a.EndTime,
		MaxParticipants:  a.MaxParticipants,
		Status:           a.Status,
		IsGlobal:         mtproto.ToBool(a.IsGlobal == 1),
		ParticipantCount: a.ParticipantCount,
		IsJoined:         mtproto.ToBool(isJoined),
		CreatorName:      a.CreatorName,
		CreatedAt:        a.CreatedAt,
	}).To_CityActivity()
}
