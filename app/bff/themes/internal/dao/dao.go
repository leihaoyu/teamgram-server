package dao

import (
	"github.com/teamgram/marmota/pkg/net/rpcx"
	"github.com/teamgram/teamgram-server/app/bff/themes/internal/config"
	msg_client "github.com/teamgram/teamgram-server/app/messenger/msg/msg/client"
	dialog_client "github.com/teamgram/teamgram-server/app/service/biz/dialog/client"
	user_client "github.com/teamgram/teamgram-server/app/service/biz/user/client"
)

type Dao struct {
	msg_client.MsgClient
	dialog_client.DialogClient
	user_client.UserClient
}

func New(c config.Config) *Dao {
	return &Dao{
		MsgClient:    msg_client.NewMsgClient(rpcx.GetCachedRpcClient(c.MsgClient)),
		DialogClient: dialog_client.NewDialogClient(rpcx.GetCachedRpcClient(c.DialogClient)),
		UserClient:   user_client.NewUserClient(rpcx.GetCachedRpcClient(c.UserClient)),
	}
}
