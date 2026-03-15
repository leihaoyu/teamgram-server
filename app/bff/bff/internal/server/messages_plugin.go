package server

import (
	"context"
	"encoding/base64"
	"hash/fnv"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/pkg/webpage"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	recentStickersLimit = 20
)

type messagesPluginImpl struct {
	db *sqlx.DB
}

func newMessagesPlugin(c sqlx.Config) *messagesPluginImpl {
	return &messagesPluginImpl{
		db: sqlx.NewMySQL(&c),
	}
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

	og, err := webpage.Fetch(rawURL)
	if err != nil {
		log.Infof("GetWebpagePreview - fetch error for %s: %v", rawURL, err)
		return nil, nil
	}

	// Must have at least a title or description
	if og.Title == "" && og.Description == "" {
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
