/*
 * Created from 'scheme.tl' by 'mtprotoc'
 *
 * Copyright (c) 2021-present,  Teamgram Studio (https://teamgram.io).
 *  All rights reserved.
 *
 * Author: teamgramio (teamgram.io@gmail.com)
 */

package dao

import (
	kafka "github.com/teamgram/marmota/pkg/mq"
	"github.com/teamgram/marmota/pkg/net/rpcx"
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	sync_client "github.com/teamgram/teamgram-server/app/messenger/sync/client"
	"github.com/teamgram/teamgram-server/app/messenger/sync/internal/config"
	chat_client "github.com/teamgram/teamgram-server/app/service/biz/chat/client"
	idgen_client "github.com/teamgram/teamgram-server/app/service/idgen/client"
	status_client "github.com/teamgram/teamgram-server/app/service/status/client"
	apns2 "github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/kv"
	"github.com/zeromicro/go-zero/zrpc"
)

type Dao struct {
	*Mysql
	kv             kv.Store
	conf           *config.Config
	sessionServers map[string]*Session
	idgen_client.IDGenClient2
	status_client.StatusClient
	chat_client.ChatClient
	PushClient   sync_client.SyncClient
	APNsClient   *apns2.Client
	APNsBundleID string
	DevicesDB    *sqlx.DB
}

func New(c config.Config) *Dao {
	db := sqlx.NewMySQL(&c.Mysql)
	d := &Dao{
		Mysql:          newMysqlDao(db),
		kv:             kv.NewStore(c.KV),
		conf:           &c,
		sessionServers: make(map[string]*Session),
		IDGenClient2:   idgen_client.NewIDGenClient2(zrpc.MustNewClient(c.IdgenClient)),
		StatusClient:   status_client.NewStatusClient(zrpc.MustNewClient(c.StatusClient)),
		ChatClient:     chat_client.NewChatClient(rpcx.GetCachedRpcClient(c.ChatClient)),
	}
	if c.PushClient != nil {
		d.PushClient = sync_client.NewSyncMqClient(kafka.MustKafkaProducer(c.PushClient))
	}
	if c.DevicesMySQL != nil {
		d.DevicesDB = sqlx.NewMySQL(c.DevicesMySQL)
	}
	if c.APNs != nil {
		authKey, err := token.AuthKeyFromFile(c.APNs.KeyFile)
		if err != nil {
			logx.Errorf("APNs: failed to load auth key from %s: %v", c.APNs.KeyFile, err)
		} else {
			tkn := &token.Token{
				AuthKey: authKey,
				KeyID:   c.APNs.KeyID,
				TeamID:  c.APNs.TeamID,
			}
			if c.APNs.Production {
				d.APNsClient = apns2.NewTokenClient(tkn).Production()
			} else {
				d.APNsClient = apns2.NewTokenClient(tkn).Development()
			}
			d.APNsBundleID = c.APNs.BundleID
			logx.Infof("APNs: client initialized, bundleID=%s, production=%v", c.APNs.BundleID, c.APNs.Production)
		}
	}

	go d.watch(c.SessionClient)
	return d
}
