package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"math"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/teamgram/marmota/pkg/net/rpcx"
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/service/biz/chat/chat"
	chat_client "github.com/teamgram/teamgram-server/app/service/biz/chat/client"
	user_client "github.com/teamgram/teamgram-server/app/service/biz/user/client"
	userpb "github.com/teamgram/teamgram-server/app/service/biz/user/user"
	username_client "github.com/teamgram/teamgram-server/app/service/biz/username/client"
	"github.com/teamgram/teamgram-server/app/service/biz/username/username"
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
	db             *sqlx.DB
	mediaClient    media_client.MediaClient
	dfsClient      dfs_client.DfsClient
	idgenClient2   idgen_client.IDGenClient2
	usernameClient username_client.UsernameClient
	userClient     user_client.UserClient
	chatClient     chat_client.ChatClient
}

func newMessagesPlugin(mysqlConf sqlx.Config, mediaConf, dfsConf, idgenConf, bizServiceConf zrpc.RpcClientConf) *messagesPluginImpl {
	bizConn := rpcx.GetCachedRpcClient(bizServiceConf)
	return &messagesPluginImpl{
		db:             sqlx.NewMySQL(&mysqlConf),
		mediaClient:    media_client.NewMediaClient(rpcx.GetCachedRpcClient(mediaConf)),
		dfsClient:      dfs_client.NewDfsClient(rpcx.GetCachedRpcClient(dfsConf)),
		idgenClient2:   idgen_client.NewIDGenClient2(rpcx.GetCachedRpcClient(idgenConf)),
		usernameClient: username_client.NewUsernameClient(bizConn),
		userClient:     user_client.NewUserClient(bizConn),
		chatClient:     chat_client.NewChatClient(bizConn),
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
// isTelegramHost — check if hostname is t.me or telegram.me
// ============================================================================

func isTelegramHost(hostname string) bool {
	h := strings.ToLower(hostname)
	return h == "t.me" || h == "telegram.me"
}

// ============================================================================
// makePageId — generate stable page ID from URL
// ============================================================================

func makePageId(rawURL string) int64 {
	h := fnv.New64a()
	h.Write([]byte(rawURL))
	return int64(h.Sum64())
}

// ============================================================================
// handleTelegramURL — resolve t.me/* internal links
// ============================================================================

func (p *messagesPluginImpl) handleTelegramURL(ctx context.Context, rawURL string, parsed *url.URL) (*mtproto.WebPage, error) {
	log := logx.WithContext(ctx)

	pathStr := strings.TrimPrefix(parsed.Path, "/")
	pathStr = strings.TrimSuffix(pathStr, "/")
	if pathStr == "" {
		return nil, nil
	}
	segments := strings.Split(pathStr, "/")

	pageId := makePageId(rawURL)
	displayUrl := parsed.Host + parsed.Path

	// t.me/addstickers/{setname}
	if len(segments) == 2 && segments[0] == "addstickers" {
		return p.buildStickerSetPage(ctx, pageId, rawURL, displayUrl, segments[1], false)
	}

	// t.me/addemoji/{setname}
	if len(segments) == 2 && segments[0] == "addemoji" {
		return p.buildStickerSetPage(ctx, pageId, rawURL, displayUrl, segments[1], true)
	}

	// t.me/+{hash} — invite link
	if len(segments) == 1 && strings.HasPrefix(segments[0], "+") {
		return p.buildInvitePage(ctx, pageId, rawURL, displayUrl)
	}

	// t.me/c/{channelId}/{msgId} — private channel message
	if len(segments) >= 3 && segments[0] == "c" {
		return p.buildMessagePage(pageId, rawURL, displayUrl), nil
	}

	// t.me/{username}/{msgId} — public channel/group message
	if len(segments) == 2 {
		if _, err := strconv.ParseInt(segments[1], 10, 64); err == nil {
			// second segment is a number -> message link
			return p.buildMessagePage(pageId, rawURL, displayUrl), nil
		}
	}

	// t.me/{username} — resolve username to user/channel/chat
	if len(segments) == 1 {
		name := segments[0]
		// skip known reserved paths
		reserved := map[string]bool{
			"bg": true, "addtheme": true, "addlist": true,
			"boost": true, "share": true, "proxy": true, "socks": true,
		}
		if reserved[strings.ToLower(name)] {
			return nil, nil
		}
		return p.buildUsernamePage(ctx, pageId, rawURL, displayUrl, name, log)
	}

	return nil, nil
}

func (p *messagesPluginImpl) buildStickerSetPage(ctx context.Context, pageId int64, rawURL, displayUrl, setName string, isEmoji bool) (*mtproto.WebPage, error) {
	wpType := "telegram_stickerset"
	title := "Stickers"
	if isEmoji {
		title = "Emoji"
	}
	wp := mtproto.MakeTLWebPage(&mtproto.WebPage{
		Id:          pageId,
		Url_STRING:  rawURL,
		DisplayUrl:  displayUrl,
		Hash:        int32(time.Now().Unix()),
		Type:        mtproto.MakeFlagsString(wpType),
		SiteName:    mtproto.MakeFlagsString("Telegram"),
		Title:       mtproto.MakeFlagsString(title),
		Description: mtproto.MakeFlagsString(setName),
		Date:        int32(time.Now().Unix()),
	}).To_WebPage()
	_ = ctx
	return wp, nil
}

func (p *messagesPluginImpl) buildInvitePage(ctx context.Context, pageId int64, rawURL, displayUrl string) (*mtproto.WebPage, error) {
	wp := mtproto.MakeTLWebPage(&mtproto.WebPage{
		Id:         pageId,
		Url_STRING: rawURL,
		DisplayUrl: displayUrl,
		Hash:       int32(time.Now().Unix()),
		Type:       mtproto.MakeFlagsString("telegram_chat_request"),
		SiteName:   mtproto.MakeFlagsString("Telegram"),
		Title:      mtproto.MakeFlagsString("Invite Link"),
		Date:       int32(time.Now().Unix()),
	}).To_WebPage()
	_ = ctx
	return wp, nil
}

func (p *messagesPluginImpl) buildMessagePage(pageId int64, rawURL, displayUrl string) *mtproto.WebPage {
	return mtproto.MakeTLWebPage(&mtproto.WebPage{
		Id:         pageId,
		Url_STRING: rawURL,
		DisplayUrl: displayUrl,
		Hash:       int32(time.Now().Unix()),
		Type:       mtproto.MakeFlagsString("telegram_message"),
		SiteName:   mtproto.MakeFlagsString("Telegram"),
		Title:      mtproto.MakeFlagsString("Message"),
		Date:       int32(time.Now().Unix()),
	}).To_WebPage()
}

func (p *messagesPluginImpl) buildUsernamePage(ctx context.Context, pageId int64, rawURL, displayUrl, name string, log logx.Logger) (*mtproto.WebPage, error) {
	peer, err := p.usernameClient.UsernameResolveUsername(ctx, &username.TLUsernameResolveUsername{
		Username: name,
	})
	if err != nil || peer == nil {
		log.Infof("handleTelegramURL - username %q not found: %v", name, err)
		return nil, nil
	}

	switch peer.GetPredicateName() {
	case mtproto.Predicate_peerUser:
		return p.buildUserPage(ctx, pageId, rawURL, displayUrl, peer.GetUserId(), log)
	case mtproto.Predicate_peerChat:
		return p.buildChatPage(ctx, pageId, rawURL, displayUrl, peer.GetChatId(), log)
	case mtproto.Predicate_peerChannel:
		return p.buildChannelPage(ctx, pageId, rawURL, displayUrl, peer.GetChannelId(), log)
	default:
		return nil, nil
	}
}

func (p *messagesPluginImpl) buildUserPage(ctx context.Context, pageId int64, rawURL, displayUrl string, userId int64, log logx.Logger) (*mtproto.WebPage, error) {
	immUser, err := p.userClient.UserGetImmutableUser(ctx, &userpb.TLUserGetImmutableUser{
		Id: userId,
	})
	if err != nil || immUser == nil || immUser.GetUser() == nil {
		log.Infof("handleTelegramURL - user %d not found: %v", userId, err)
		return nil, nil
	}
	ud := immUser.GetUser()
	displayName := strings.TrimSpace(ud.GetFirstName() + " " + ud.GetLastName())
	if displayName == "" {
		displayName = ud.GetUsername()
	}

	desc := ""
	if ud.GetAbout() != nil {
		desc = ud.GetAbout().GetValue()
	}

	wp := mtproto.MakeTLWebPage(&mtproto.WebPage{
		Id:         pageId,
		Url_STRING: rawURL,
		DisplayUrl: displayUrl,
		Hash:       int32(time.Now().Unix()),
		Type:       mtproto.MakeFlagsString("telegram_user"),
		SiteName:   mtproto.MakeFlagsString("Telegram"),
		Title:      mtproto.MakeFlagsString(displayName),
		Date:       int32(time.Now().Unix()),
	}).To_WebPage()

	if desc != "" {
		wp.Description = mtproto.MakeFlagsString(desc)
	}

	return wp, nil
}

func (p *messagesPluginImpl) buildChatPage(ctx context.Context, pageId int64, rawURL, displayUrl string, chatId int64, log logx.Logger) (*mtproto.WebPage, error) {
	mutableChat, err := p.chatClient.ChatGetMutableChat(ctx, &chat.TLChatGetMutableChat{
		ChatId: chatId,
	})
	if err != nil || mutableChat == nil || mutableChat.GetChat() == nil {
		log.Infof("handleTelegramURL - chat %d not found: %v", chatId, err)
		return nil, nil
	}
	chatData := mutableChat.GetChat()

	desc := chatData.GetAbout()
	if desc == "" {
		desc = fmt.Sprintf("%d members", chatData.GetParticipantsCount())
	}

	wp := mtproto.MakeTLWebPage(&mtproto.WebPage{
		Id:          pageId,
		Url_STRING:  rawURL,
		DisplayUrl:  displayUrl,
		Hash:        int32(time.Now().Unix()),
		Type:        mtproto.MakeFlagsString("telegram_chat"),
		SiteName:    mtproto.MakeFlagsString("Telegram"),
		Title:       mtproto.MakeFlagsString(chatData.GetTitle()),
		Description: mtproto.MakeFlagsString(desc),
		Date:        int32(time.Now().Unix()),
	}).To_WebPage()

	return wp, nil
}

func (p *messagesPluginImpl) buildChannelPage(ctx context.Context, pageId int64, rawURL, displayUrl string, channelId int64, log logx.Logger) (*mtproto.WebPage, error) {
	// In teamgram, channels are also stored as chats
	mutableChat, err := p.chatClient.ChatGetMutableChat(ctx, &chat.TLChatGetMutableChat{
		ChatId: channelId,
	})
	if err != nil || mutableChat == nil || mutableChat.GetChat() == nil {
		log.Infof("handleTelegramURL - channel %d not found: %v", channelId, err)
		return nil, nil
	}
	chatData := mutableChat.GetChat()

	// TODO: distinguish channel vs megagroup if chat metadata provides that info
	wpType := "telegram_channel"

	desc := chatData.GetAbout()
	if desc == "" {
		desc = fmt.Sprintf("%d members", chatData.GetParticipantsCount())
	}

	wp := mtproto.MakeTLWebPage(&mtproto.WebPage{
		Id:          pageId,
		Url_STRING:  rawURL,
		DisplayUrl:  displayUrl,
		Hash:        int32(time.Now().Unix()),
		Type:        mtproto.MakeFlagsString(wpType),
		SiteName:    mtproto.MakeFlagsString("Telegram"),
		Title:       mtproto.MakeFlagsString(chatData.GetTitle()),
		Description: mtproto.MakeFlagsString(desc),
		Date:        int32(time.Now().Unix()),
	}).To_WebPage()

	return wp, nil
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

	// t.me / telegram.me internal links — route to internal resolver
	if isTelegramHost(parsed.Hostname()) {
		return p.handleTelegramURL(ctx, rawURL, parsed)
	}

	// Generate a stable ID from URL
	pageId := makePageId(rawURL)

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
			Id:         pageId,
			Url_STRING: rawURL,
			DisplayUrl: displayUrl,
			Hash:       int32(time.Now().Unix()),
			Type:       mtproto.MakeFlagsString("photo"),
			SiteName:   mtproto.MakeFlagsString(parsed.Host),
			Title:      mtproto.MakeFlagsString(path.Base(parsed.Path)),
			Photo:      photo,
			Date:       int32(time.Now().Unix()),
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

	// Embed fields (YouTube, Vimeo, etc.)
	if og.EmbedURL != "" {
		wp.EmbedUrl = mtproto.MakeFlagsString(og.EmbedURL)
		if og.EmbedType != "" {
			wp.EmbedType = mtproto.MakeFlagsString(og.EmbedType)
		}
		if og.EmbedWidth != "" {
			if w, err := strconv.Atoi(og.EmbedWidth); err == nil {
				wp.EmbedWidth = mtproto.MakeFlagsInt32(int32(w))
			}
		}
		if og.EmbedHeight != "" {
			if h, err := strconv.Atoi(og.EmbedHeight); err == nil {
				wp.EmbedHeight = mtproto.MakeFlagsInt32(int32(h))
			}
		}
	}

	// Author
	if og.Author != "" {
		wp.Author = mtproto.MakeFlagsString(og.Author)
	}

	// If og:image is present, download and attach as Photo (graceful degradation on failure)
	imageSource := og.Image
	if imageSource == "" && og.Favicon != "" {
		imageSource = og.Favicon // fallback to favicon for small icon display
	}
	if imageSource != "" {
		imageURL := webpage.ResolveImageURL(rawURL, imageSource)
		// SSRF check on resolved image URL
		_, imgParsed, imgErr := webpage.NormalizeURL(imageURL)
		if imgErr == nil && !webpage.IsPrivateHost(imgParsed.Hostname()) {
			photo := p.downloadAndUploadPhoto(ctx, imageURL)
			if photo != nil {
				wp.Photo = photo
				// favicon as sole source → keep small (article default); og:image → large
				if og.Image != "" {
					wp.HasLargeMedia = true
				}
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
