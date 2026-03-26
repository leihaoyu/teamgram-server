package grpc

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/server/grpc/service"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/svc"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

func New(svcCtx *svc.ServiceContext, c zrpc.RpcServerConf) *zrpc.RpcServer {
	s := zrpc.MustNewServer(c, func(grpcServer *grpc.Server) {
		svc := service.New(svcCtx)
		mtproto.RegisterRPCStickersServer(grpcServer, svc)
		mtproto.RegisterRPCCustomEmojisServer(grpcServer, svc)
	})
	return s
}
