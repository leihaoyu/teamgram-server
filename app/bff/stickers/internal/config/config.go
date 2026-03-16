package config

import (
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	TelegramBotToken    string
	FeaturedStickerSets []string `json:",optional"`
	Mysql               sqlx.Config
	IdgenClient         zrpc.RpcClientConf
	MediaClient         zrpc.RpcClientConf
	DfsClient           zrpc.RpcClientConf
}
