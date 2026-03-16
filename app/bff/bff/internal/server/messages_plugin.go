package server

import (
	"context"
	"encoding/base64"
	"hash/fnv"
	"math"
	"net/url"
	"path"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/teamgram/marmota/pkg/net/rpcx"
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/proto/mtproto"
	dfs_client "github.com/teamgram/teamgram-server/app/service/dfs/client"
	"github.com/teamgram/teamgram-server/app/service/dfs/dfs"
	idgen_client "github.com/teamgram/teamgram-server/app/service/idgen/client"
	media_client "github.com/teamgram/teamgram-server/app/service/media/client"
	mediapb "github.com/teamgram/teamgram-server/app/service/media/media"
	"github.com/teamgram/teamgram-server/pkg/webpage"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

const (
	recentStickersLimit = 20
	filePartSize        = 512 * 1024 // 512KB per upload part
)

type messagesPluginImpl struct {
	db           *sqlx.DB
	mediaClient  media_client.MediaClient
	dfsClient    dfs_client.DfsClient
	idgenClient2 idgen_client.IDGenClient2
}

func newMessagesPlugin(mysqlConf sqlx.Config, mediaConf, dfsConf, idgenConf zrpc.RpcClientConf) *messagesPluginImpl {
	return &messagesPluginImpl{
		db:           sqlx.NewMySQL(&mysqlConf),
		mediaClient:  media_client.NewMediaClient(rpcx.GetCachedRpcClient(mediaConf)),
		dfsClient:    dfs_client.NewDfsClient(rpcx.GetCachedRpcClient(dfsConf)),
		idgenClient2: idgen_client.NewIDGenClient2(rpcx.GetCachedRpcClient(idgenConf)),
	}
}

// ============================================================================
// uploadPhotoData — upload pre-downloaded image bytes to DFS, return Photo
// ============================================================================

func (p *messagesPluginImpl) uploadPhotoData(ctx context.Context, data []byte, ext string) *mtproto.Photo {
	log := logx.WithContext(ctx)

	if len(data) == 0 {
		return nil
	}

	tempFileId := p.idgenClient2.NextId(ctx)
	if tempFileId == 0 {
		log.Errorf("uploadPhotoData - idgen returned 0")
		return nil
	}

	totalParts := int32(math.Ceil(float64(len(data)) / float64(filePartSize)))
	if totalParts == 0 {
		totalParts = 1
	}

	for part := int32(0); part < totalParts; part++ {
		start := int(part) * filePartSize
		end := start + filePartSize
		if end > len(data) {
			end = len(data)
		}
		_, err := p.dfsClient.DfsWriteFilePartData(ctx, &dfs.TLDfsWriteFilePartData{
			Creator:        tempFileId,
			FileId:         tempFileId,
			FilePart:       part,
			Bytes:          data[start:end],
			Big:            false,
			FileTotalParts: &types.Int32Value{Value: totalParts},
		})
		if err != nil {
			log.Errorf("uploadPhotoData - DfsWriteFilePartData(part=%d) error: %v", part, err)
			return nil
		}
	}

	if ext == "" || len(ext) > 5 {
		ext = ".jpg"
	}

	photo, err := p.mediaClient.MediaUploadPhotoFile(ctx, &mediapb.TLMediaUploadPhotoFile{
		OwnerId: 0,
		File: mtproto.MakeTLInputFile(&mtproto.InputFile{
			Id:    tempFileId,
			Parts: totalParts,
			Name:  "webpage_preview" + ext,
		}).To_InputFile(),
	})
	if err != nil {
		log.Errorf("uploadPhotoData - MediaUploadPhotoFile error: %v", err)
		return nil
	}

	return photo
}

// downloadAndUploadPhoto downloads an image from URL and uploads to DFS.
func (p *messagesPluginImpl) downloadAndUploadPhoto(ctx context.Context, imageURL string) *mtproto.Photo {
	log := logx.WithContext(ctx)

	data, _, err := webpage.DownloadImage(imageURL)
	if err != nil {
		log.Infof("downloadAndUploadPhoto - download error for %s: %v", imageURL, err)
		return nil
	}

	// Extract extension from URL path (ignoring query string)
	parsed, _ := url.Parse(imageURL)
	ext := ""
	if parsed != nil {
		ext = path.Ext(parsed.Path)
	}
	return p.uploadPhotoData(ctx, data, ext)
}

// ============================================================================
// GetWebpagePreview — fetch URL and extract OG meta tags to build a WebPage
// ============================================================================

func (p *messagesPluginImpl) GetWebpagePreview(ctx context.Context, rawURL string) (*mtproto.WebPage, error) {
	log := logx.WithContext(ctx)

	// Normalize URL
	rawURL, parsed, err := webpage.NormalizeURL(rawURL)
	if err != nil {
		log.Infof("GetWebpagePreview - invalid URL: %s", rawURL)
		return nil, nil
	}
	// Block private/loopback IPs
	if webpage.IsPrivateHost(parsed.Hostname()) {
		return nil, nil
	}

	// Generate a stable ID from URL
	h := fnv.New64a()
	h.Write([]byte(rawURL))
	pageId := int64(h.Sum64())

	displayUrl := parsed.Host + parsed.Path
	if len(displayUrl) > 80 {
		displayUrl = displayUrl[:80] + "..."
	}

	// Fetch page (handles both HTML and direct image URLs)
	og, err := webpage.Fetch(rawURL)
	if err != nil {
		log.Infof("GetWebpagePreview - fetch error for %s: %v", rawURL, err)
		return nil, nil
	}

	// Scenario B: Fetch() detected a direct image URL (Content-Type: image/*)
	if len(og.ImageData) > 0 {
		ext := path.Ext(parsed.Path)
		photo := p.uploadPhotoData(ctx, og.ImageData, ext)
		if photo == nil {
			return nil, nil
		}
		wp := mtproto.MakeTLWebPage(&mtproto.WebPage{
			Id:          pageId,
			Url_STRING:  rawURL,
			DisplayUrl:  displayUrl,
			Hash:        int32(time.Now().Unix()),
			Type:        mtproto.MakeFlagsString("photo"),
			SiteName:    mtproto.MakeFlagsString(parsed.Host),
			Title:       mtproto.MakeFlagsString(path.Base(parsed.Path)),
			Photo:       photo,
			Date:        int32(time.Now().Unix()),
		}).To_WebPage()
		return wp, nil
	}

	// Scenario A: Normal webpage with OG meta
	if og.Title == "" && og.Description == "" {
		return nil, nil
	}

	pageType := og.Type
	if pageType == "" {
		pageType = "article"
	}

	wp := mtproto.MakeTLWebPage(&mtproto.WebPage{
		Id:          pageId,
		Url_STRING:  rawURL,
		DisplayUrl:  displayUrl,
		Hash:        int32(time.Now().Unix()),
		Type:        mtproto.MakeFlagsString(pageType),
		SiteName:    mtproto.MakeFlagsString(og.SiteName),
		Title:       mtproto.MakeFlagsString(og.Title),
		Description: mtproto.MakeFlagsString(og.Description),
		Date:        int32(time.Now().Unix()),
	}).To_WebPage()

	// If og:image is present, download and attach as Photo (graceful degradation on failure)
	if og.Image != "" {
		imageURL := webpage.ResolveImageURL(rawURL, og.Image)
		// SSRF check on resolved image URL
		_, imgParsed, imgErr := webpage.NormalizeURL(imageURL)
		if imgErr == nil && !webpage.IsPrivateHost(imgParsed.Hostname()) {
			photo := p.downloadAndUploadPhoto(ctx, imageURL)
			if photo != nil {
				wp.Photo = photo
			}
		}
	}

	return wp, nil
}

// ============================================================================
// GetMessageMedia — not implemented
// ============================================================================

func (p *messagesPluginImpl) GetMessageMedia(ctx context.Context, ownerId int64, media *mtproto.InputMedia) (*mtproto.MessageMedia, error) {
	return nil, nil
}

// ============================================================================
// SaveRecentSticker — auto-save sticker to user's recent list
// ============================================================================

func (p *messagesPluginImpl) SaveRecentSticker(ctx context.Context, userId int64, doc *mtproto.Document) {
	if doc == nil {
		return
	}
	log := logx.WithContext(ctx)

	// 1. Extract emoji from documentAttributeSticker
	emoji := ""
	for _, attr := range doc.GetAttributes() {
		if attr.GetPredicateName() == mtproto.Predicate_documentAttributeSticker {
			emoji = attr.GetAlt()
			break
		}
	}

	// 2. Serialize Document to base64
	data, err := proto.Marshal(doc)
	if err != nil {
		log.Errorf("SaveRecentSticker - proto.Marshal error: %v", err)
		return
	}
	docData := base64.StdEncoding.EncodeToString(data)

	// 3. Upsert (UNIQUE KEY (user_id, document_id) prevents duplicates)
	upsertQuery := "INSERT INTO user_recent_stickers(user_id, document_id, emoji, document_data, date2) " +
		"VALUES (?, ?, ?, ?, ?) " +
		"ON DUPLICATE KEY UPDATE document_data = VALUES(document_data), emoji = VALUES(emoji), date2 = VALUES(date2), deleted = 0"
	_, err = p.db.Exec(ctx, upsertQuery, userId, doc.GetId(), emoji, docData, time.Now().Unix())
	if err != nil {
		log.Errorf("SaveRecentSticker - upsert error: %v", err)
		return
	}

	// 4. Trim to 20 entries — soft-delete oldest beyond limit
	trimQuery := "UPDATE user_recent_stickers SET deleted = 1 " +
		"WHERE user_id = ? AND deleted = 0 " +
		"AND id NOT IN (" +
		"  SELECT id FROM (SELECT id FROM user_recent_stickers WHERE user_id = ? AND deleted = 0 ORDER BY date2 DESC LIMIT ?" +
		"  ) AS keep)"
	_, err = p.db.Exec(ctx, trimQuery, userId, userId, recentStickersLimit)
	if err != nil {
		log.Errorf("SaveRecentSticker - trim error: %v", err)
	}
}
