package dao

import (
	"sync"

	"github.com/teamgram/marmota/pkg/net/rpcx"
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/config"
	dfs_client "github.com/teamgram/teamgram-server/app/service/dfs/client"
	idgen_client "github.com/teamgram/teamgram-server/app/service/idgen/client"
	media_client "github.com/teamgram/teamgram-server/app/service/media/client"
)

// globalDownloadSem limits total concurrent sticker file downloads across all requests.
// This prevents memory explosion when many sticker sets are fetched simultaneously.
var globalDownloadSem = make(chan struct{}, 2)

// singleFlightCall is a simple singleflight implementation for sticker set fetches.
type singleFlightCall struct {
	wg  sync.WaitGroup
	err error
}

// StickerSetFlight deduplicates concurrent fetchAndCacheStickerSet calls for the same shortName.
type StickerSetFlight struct {
	mu    sync.Mutex
	calls map[string]*singleFlightCall
}

func NewStickerSetFlight() *StickerSetFlight {
	return &StickerSetFlight{
		calls: make(map[string]*singleFlightCall),
	}
}

// Do ensures only one fetch runs for a given key. Returns (didRun, error).
// If another goroutine is already fetching this key, waits and returns (false, thatError).
func (f *StickerSetFlight) Do(key string, fn func() error) (bool, error) {
	f.mu.Lock()
	if call, ok := f.calls[key]; ok {
		f.mu.Unlock()
		call.wg.Wait()
		return false, call.err
	}
	call := &singleFlightCall{}
	call.wg.Add(1)
	f.calls[key] = call
	f.mu.Unlock()

	call.err = fn()
	call.wg.Done()

	f.mu.Lock()
	delete(f.calls, key)
	f.mu.Unlock()

	return true, call.err
}

type Dao struct {
	*Mysql
	idgen_client.IDGenClient2
	media_client.MediaClient
	dfs_client.DfsClient
	BotAPI          *BotAPIClient
	StickerSetFetch *StickerSetFlight
}

func New(c config.Config) *Dao {
	db := sqlx.NewMySQL(&c.Mysql)
	return &Dao{
		Mysql:           newMysqlDao(db),
		IDGenClient2:    idgen_client.NewIDGenClient2(rpcx.GetCachedRpcClient(c.IdgenClient)),
		MediaClient:     media_client.NewMediaClient(rpcx.GetCachedRpcClient(c.MediaClient)),
		DfsClient:       dfs_client.NewDfsClient(rpcx.GetCachedRpcClient(c.DfsClient)),
		BotAPI:          NewBotAPIClient(c.TelegramBotToken),
		StickerSetFetch: NewStickerSetFlight(),
	}
}
