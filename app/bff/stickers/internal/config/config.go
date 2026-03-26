package config

import (
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type MinioConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
}

type Config struct {
	zrpc.RpcServerConf

	TelegramBotToken         string
	FeaturedStickerSets      []string `json:",optional"`
	FeaturedEmojiStickerSets []string `json:",optional"`
	Mysql                    sqlx.Config
	Minio                    MinioConfig
	IdgenClient              zrpc.RpcClientConf
	MediaClient              zrpc.RpcClientConf
	DfsClient                zrpc.RpcClientConf
}
