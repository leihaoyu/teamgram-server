package core

import (
	"github.com/teamgram/proto/mtproto"
)

func (c *CityActivityCore) CityActivityJoinActivity(in *mtproto.TLCityActivityJoinActivity) (*mtproto.Bool, error) {
	if c.MD == nil {
		return mtproto.BoolFalse, mtproto.ErrInternelServerError
	}

	city := in.GetCity()
	if city == "" && c.MD.ClientAddr != "" {
		city = c.svcCtx.Dao.GetCityByIp(c.MD.ClientAddr)
	}

	err := c.svcCtx.Dao.JoinActivity(c.ctx, in.GetId(), c.MD.UserId, city)
	if err != nil {
		c.Logger.Errorf("cityActivity.joinActivity - error: %v", err)
		return mtproto.BoolFalse, err
	}
	return mtproto.BoolTrue, nil
}
