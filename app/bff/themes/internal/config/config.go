package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	MsgClient    zrpc.RpcClientConf
	DialogClient zrpc.RpcClientConf
	UserClient   zrpc.RpcClientConf
}
