package stickers_helper

import (
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/config"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/server/grpc/service"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/svc"
)

type (
	Config      = config.Config
	MinioConfig = config.MinioConfig
)

func New(c Config) *service.Service {
	return service.New(svc.NewServiceContext(c))
}
