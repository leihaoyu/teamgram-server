package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// maxFileDownloadSize caps file downloads to prevent unbounded memory allocation.
	maxFileDownloadSize = 10 * 1024 * 1024 // 10 MB
)

// BotAPIClient is a lightweight HTTP client for Telegram Bot API
type BotAPIClient struct {
	token      string
	apiClient  *http.Client // for API calls (getFile, getStickerSet)
	fileClient *http.Client // for file downloads (longer timeout)
}

func NewBotAPIClient(token string) *BotAPIClient {
	// Shared transport with connection pooling for concurrent downloads
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &BotAPIClient{
		token: token,
		apiClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		fileClient: &http.Client{
			Timeout:   120 * time.Second, // file downloads need more time
			Transport: transport,
		},
	}
}

// Bot API response types

type BotAPIStickerSetResponse struct {
	Ok          bool              `json:"ok"`
	Result      *BotAPIStickerSet `json:"result"`
	Description string            `json:"description"`
}

type BotAPIStickerSet struct {
	Name        string           `json:"name"`
	Title       string           `json:"title"`
	StickerType string           `json:"sticker_type"`
	Stickers    []BotAPISticker  `json:"stickers"`
	Thumbnail   *BotAPIPhotoSize `json:"thumbnail,omitempty"`
}

type BotAPISticker struct {
	FileId           string           `json:"file_id"`
	FileUniqueId     string           `json:"file_unique_id"`
	FileSize         int64            `json:"file_size"`
	Width            int32            `json:"width"`
	Height           int32            `json:"height"`
	Emoji            string           `json:"emoji"`
	SetName          string           `json:"set_name"`
	IsAnimated       bool             `json:"is_animated"`
	IsVideo          bool             `json:"is_video"`
	Type             string           `json:"type"`
	Thumbnail        *BotAPIPhotoSize `json:"thumbnail,omitempty"`
	PremiumAnimation *BotAPIFile      `json:"premium_animation,omitempty"`
}

type BotAPIPhotoSize struct {
	FileId       string `json:"file_id"`
	FileUniqueId string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size"`
	Width        int32  `json:"width"`
	Height       int32  `json:"height"`
}

type BotAPIFileResponse struct {
	Ok          bool        `json:"ok"`
	Result      *BotAPIFile `json:"result"`
	Description string      `json:"description"`
}

type BotAPIFile struct {
	FileId       string `json:"file_id"`
	FileUniqueId string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size"`
	FilePath     string `json:"file_path"`
}

// GetStickerSet calls the Bot API getStickerSet method
func (b *BotAPIClient) GetStickerSet(ctx context.Context, name string) (*BotAPIStickerSet, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getStickerSet?name=%s",
		b.token, url.QueryEscape(name))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("botapi: create request error: %w", err)
	}

	resp, err := b.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("botapi: getStickerSet request error: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body) // drain to allow connection reuse
		resp.Body.Close()
	}()

	var result BotAPIStickerSetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("botapi: decode getStickerSet response error: %w", err)
	}

	if !result.Ok {
		logx.WithContext(ctx).Errorf("botapi: getStickerSet failed: %s", result.Description)
		return nil, fmt.Errorf("botapi: getStickerSet failed: %s", result.Description)
	}

	return result.Result, nil
}

// GetFile calls the Bot API getFile method
func (b *BotAPIClient) GetFile(ctx context.Context, fileId string) (*BotAPIFile, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s",
		b.token, url.QueryEscape(fileId))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("botapi: create request error: %w", err)
	}

	resp, err := b.apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("botapi: getFile request error: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	var result BotAPIFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("botapi: decode getFile response error: %w", err)
	}

	if !result.Ok {
		return nil, fmt.Errorf("botapi: getFile failed: %s", result.Description)
	}

	return result.Result, nil
}

// DownloadFile downloads a file from the Bot API file server.
// Caps download at maxFileDownloadSize to prevent unbounded memory allocation.
func (b *BotAPIClient) DownloadFile(ctx context.Context, filePath string) ([]byte, error) {
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.token, filePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("botapi: create download request error: %w", err)
	}

	resp, err := b.fileClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("botapi: download file error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body) // drain to allow connection reuse
		return nil, fmt.Errorf("botapi: download file returned status %d", resp.StatusCode)
	}

	// Use LimitReader to prevent unbounded memory allocation
	limited := io.LimitReader(resp.Body, maxFileDownloadSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("botapi: read file body error: %w", err)
	}
	if len(data) > maxFileDownloadSize {
		return nil, fmt.Errorf("botapi: file too large (>%d bytes)", maxFileDownloadSize)
	}

	return data, nil
}

// DownloadFileStream opens a streaming download from the Bot API file server.
// Returns the response body as io.ReadCloser — caller MUST close it.
// Also returns content length (-1 if unknown).
func (b *BotAPIClient) DownloadFileStream(ctx context.Context, filePath string) (io.ReadCloser, int64, error) {
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", b.token, filePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("botapi: create download request error: %w", err)
	}

	resp, err := b.fileClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("botapi: download file error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, 0, fmt.Errorf("botapi: download file returned status %d", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}
