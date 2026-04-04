package config

import (
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Mysql        *sqlx.Config `json:",optional"`
	MediaClient  zrpc.RpcClientConf
	ChatClient   zrpc.RpcClientConf
	UserClient   zrpc.RpcClientConf
	TestCityName string `json:",optional"`
}
