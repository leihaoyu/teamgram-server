package dao

import (
	"golang.org/x/sync/singleflight"

	"github.com/teamgram/marmota/pkg/net/rpcx"
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/config"
	dfs_client "github.com/teamgram/teamgram-server/app/service/dfs/client"
	idgen_client "github.com/teamgram/teamgram-server/app/service/idgen/client"
	media_client "github.com/teamgram/teamgram-server/app/service/media/client"
)

type Dao struct {
	*Mysql
	idgen_client.IDGenClient2
	media_client.MediaClient
	dfs_client.DfsClient
	BotAPI     *BotAPIClient
	FetchGroup singleflight.Group // deduplicates concurrent downloads of the same sticker set
}

func New(c config.Config) *Dao {
	db := sqlx.NewMySQL(&c.Mysql)
	return &Dao{
		Mysql:        newMysqlDao(db),
		IDGenClient2: idgen_client.NewIDGenClient2(rpcx.GetCachedRpcClient(c.IdgenClient)),
		MediaClient:  media_client.NewMediaClient(rpcx.GetCachedRpcClient(c.MediaClient)),
		DfsClient:    dfs_client.NewDfsClient(rpcx.GetCachedRpcClient(c.DfsClient)),
		BotAPI:       NewBotAPIClient(c.TelegramBotToken),
	}
}
