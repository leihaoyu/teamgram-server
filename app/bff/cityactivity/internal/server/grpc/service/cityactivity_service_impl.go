package service

import (
	"context"

	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/cityactivity/internal/core"
)

func (s *Service) CityActivityGetActivities(ctx context.Context, request *mtproto.TLCityActivityGetActivities) (*mtproto.CityActivity_Activities, error) {
	c := core.New(ctx, s.svcCtx)
	r, err := c.CityActivityGetActivities(request)
	return r, err
}

func (s *Service) CityActivityGetActivity(ctx context.Context, request *mtproto.TLCityActivityGetActivity) (*mtproto.CityActivity, error) {
	c := core.New(ctx, s.svcCtx)
	r, err := c.CityActivityGetActivity(request)
	return r, err
}

func (s *Service) CityActivityGetMyActivities(ctx context.Context, request *mtproto.TLCityActivityGetMyActivities) (*mtproto.CityActivity_Activities, error) {
	c := core.New(ctx, s.svcCtx)
	r, err := c.CityActivityGetMyActivities(request)
	return r, err
}

func (s *Service) CityActivityCreateActivity(ctx context.Context, request *mtproto.TLCityActivityCreateActivity) (*mtproto.CityActivity, error) {
	c := core.New(ctx, s.svcCtx)
	r, err := c.CityActivityCreateActivity(request)
	return r, err
}

func (s *Service) CityActivityEditActivity(ctx context.Context, request *mtproto.TLCityActivityEditActivity) (*mtproto.CityActivity, error) {
	c := core.New(ctx, s.svcCtx)
	r, err := c.CityActivityEditActivity(request)
	return r, err
}

func (s *Service) CityActivityDeleteActivity(ctx context.Context, request *mtproto.TLCityActivityDeleteActivity) (*mtproto.Bool, error) {
	c := core.New(ctx, s.svcCtx)
	r, err := c.CityActivityDeleteActivity(request)
	return r, err
}

func (s *Service) CityActivityJoinActivity(ctx context.Context, request *mtproto.TLCityActivityJoinActivity) (*mtproto.Bool, error) {
	c := core.New(ctx, s.svcCtx)
	r, err := c.CityActivityJoinActivity(request)
	return r, err
}

func (s *Service) CityActivityLeaveActivity(ctx context.Context, request *mtproto.TLCityActivityLeaveActivity) (*mtproto.Bool, error) {
	c := core.New(ctx, s.svcCtx)
	r, err := c.CityActivityLeaveActivity(request)
	return r, err
}
