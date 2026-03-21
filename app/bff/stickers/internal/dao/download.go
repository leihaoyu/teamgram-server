package dao

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"path"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/minio/minio-go/v7"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/pkg/imaging"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	downloadWorkers = 5  // concurrent downloads per batch
	downloadBatch   = 10 // process stickers in small batches to limit memory
)

// StickerDownloadInput holds the info needed to download one sticker file and upload it to DFS.
type StickerDownloadInput struct {
	BotFileId       string
	BotFileUniqueId string
	MimeType        string
	Attributes      []*mtproto.DocumentAttribute
	ThumbFileId     string // Bot API thumbnail file_id (optional)
	ThumbWidth      int32
	ThumbHeight     int32
}

// DownloadAndUploadStickerFiles downloads sticker files from Telegram Bot API and uploads them
// to MinIO directly via streaming. Returns Documents in the same order as inputs.
// If any file fails, returns an error (caller should not cache partial results).
// For large sets, processes in batches to limit peak memory usage.
func (d *Dao) DownloadAndUploadStickerFiles(ctx context.Context, inputs []StickerDownloadInput) ([]*mtproto.Document, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	log := logx.WithContext(ctx)
	startAll := time.Now()
	log.Infof("DownloadAndUploadStickerFiles - start: %d stickers, workers=%d, batchSize=%d", len(inputs), downloadWorkers, downloadBatch)

	results := make([]*mtproto.Document, len(inputs))

	// Process in batches to limit peak memory
	for batchStart := 0; batchStart < len(inputs); batchStart += downloadBatch {
		batchEnd := batchStart + downloadBatch
		if batchEnd > len(inputs) {
			batchEnd = len(inputs)
		}

		batch := inputs[batchStart:batchEnd]
		log.Infof("DownloadAndUploadStickerFiles - batch [%d, %d) of %d", batchStart, batchEnd, len(inputs))

		var (
			mu       sync.Mutex
			firstErr error
		)

		sem := make(chan struct{}, downloadWorkers)
		var wg sync.WaitGroup

		for i := range batch {
			idx := batchStart + i
			input := batch[i]

			wg.Add(1)
			sem <- struct{}{} // per-request concurrency limit
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() {
					if r := recover(); r != nil {
						mu.Lock()
						if firstErr == nil {
							firstErr = fmt.Errorf("panic downloading sticker %d: %v", idx, r)
						}
						mu.Unlock()
						logx.WithContext(ctx).Errorf("downloadAndUploadOne panic: %v", r)
					}
				}()

				// Acquire global semaphore to cap total memory usage across all requests
				globalDownloadSem <- struct{}{}
				defer func() { <-globalDownloadSem }()

				doc, err := d.downloadAndUploadOne(ctx, &input)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("sticker[%d] (%s): %w", idx, input.BotFileId, err)
					}
				} else {
					results[idx] = doc
				}
			}()
		}

		wg.Wait()

		if firstErr != nil {
			log.Errorf("DownloadAndUploadStickerFiles - batch FAILED after %v: %v", time.Since(startAll), firstErr)
			return nil, firstErr
		}

		// Force GC between batches to reclaim memory from completed downloads
		if batchEnd < len(inputs) {
			runtime.GC()
			debug.FreeOSMemory()
		}
	}

	log.Infof("DownloadAndUploadStickerFiles - SUCCESS: %d stickers in %v", len(inputs), time.Since(startAll))
	return results, nil
}

// downloadAndUploadOne downloads a single sticker file from Bot API and streams it directly
// to MinIO, bypassing the Media→DFS gRPC chain entirely. This avoids multiple in-memory
// copies of the file data.
func (d *Dao) downloadAndUploadOne(ctx context.Context, input *StickerDownloadInput) (*mtproto.Document, error) {
	log := logx.WithContext(ctx)
	start := time.Now()

	// 1. Generate document ID
	documentId := d.IDGenClient2.NextId(ctx)

	// 2. Compute access hash (same formula as DFS handler)
	ext := path.Ext(input.BotFileUniqueId + stickerExtForMime(input.MimeType))
	if ext == "" {
		ext = ".dat"
	}
	extType := getStorageFileTypeConstructor(ext)
	accessHash := int64(extType)<<32 | int64(rand.Uint32())

	// 3. Get file path from Bot API
	fileInfo, err := d.BotAPI.GetFile(ctx, input.BotFileId)
	if err != nil {
		return nil, fmt.Errorf("GetFile: %w", err)
	}
	tGetFile := time.Since(start)

	// 4. Stream download directly to MinIO (zero-copy, no []byte buffer)
	body, contentLength, err := d.BotAPI.DownloadFileStream(ctx, fileInfo.FilePath)
	if err != nil {
		return nil, fmt.Errorf("DownloadFileStream: %w", err)
	}
	defer body.Close()

	minioPath := fmt.Sprintf("%d.dat", documentId)
	uploadInfo, err := d.MinIO.Client.PutObject(
		ctx,
		"documents",
		minioPath,
		body,
		contentLength, // pass content length if known (-1 if unknown)
		minio.PutObjectOptions{ContentType: input.MimeType},
	)
	if err != nil {
		return nil, fmt.Errorf("MinIO PutObject: %w", err)
	}
	tUpload := time.Since(start)
	fileSize := uploadInfo.Size

	// 5. Handle thumbnail (small enough to buffer in memory)
	var thumbs []*mtproto.PhotoSize
	if input.ThumbFileId != "" {
		thumbData, thumbErr := d.downloadThumbBytes(ctx, input.ThumbFileId)
		if thumbErr == nil && len(thumbData) > 0 {
			// Store thumbnail to MinIO photos bucket
			thumbPath := fmt.Sprintf("m/%d.dat", documentId)
			_, thumbErr = d.MinIO.Client.PutObject(
				ctx,
				"photos",
				thumbPath,
				bytes.NewReader(thumbData),
				int64(len(thumbData)),
				minio.PutObjectOptions{ContentType: "image/webp"},
			)
			if thumbErr == nil {
				thumbs = []*mtproto.PhotoSize{
					mtproto.MakeTLPhotoSize(&mtproto.PhotoSize{
						Type:  "m",
						W:     input.ThumbWidth,
						H:     input.ThumbHeight,
						Size2: int32(len(thumbData)),
					}).To_PhotoSize(),
				}

				// Generate PhotoStrippedSize from the thumbnail data
				// Sticker thumbnails are WebP, use DecodeWebp explicitly
				thumbImg, decErr := imaging.DecodeWebp(bytes.NewReader(thumbData))
				if decErr != nil {
					// Fallback to generic decode for non-WebP thumbnails
					thumbImg, decErr = imaging.Decode(bytes.NewReader(thumbData))
				}
				if decErr == nil {
					var resized = thumbImg
					if thumbImg.Bounds().Dx() >= thumbImg.Bounds().Dy() {
						resized = imaging.Resize(thumbImg, 40, 0)
					} else {
						resized = imaging.Resize(thumbImg, 0, 40)
					}
					stripped := &bytes.Buffer{}
					if encErr := imaging.EncodeStripped(stripped, resized, 30); encErr == nil {
						// Prepend stripped size before the "m" PhotoSize
						thumbs = append([]*mtproto.PhotoSize{
							mtproto.MakeTLPhotoStrippedSize(&mtproto.PhotoSize{
								Type:  "i",
								Bytes: stripped.Bytes(),
							}).To_PhotoSize(),
						}, thumbs...)
					}
				}
			} else {
				log.Infof("downloadAndUploadOne - thumb upload failed (non-fatal): %v", thumbErr)
			}
		} else if thumbErr != nil {
			log.Infof("downloadAndUploadOne - thumb download failed (non-fatal): %v", thumbErr)
		}
	}

	// 6. Build Document proto (same structure as DFS handler)
	document := mtproto.MakeTLDocument(&mtproto.Document{
		Id:            documentId,
		AccessHash:    accessHash,
		FileReference: []byte{},
		Date:          int32(time.Now().Unix()),
		MimeType:      input.MimeType,
		Size2_INT32:   int32(fileSize),
		Size2_INT64:   fileSize,
		Thumbs:        thumbs,
		VideoThumbs:   nil,
		DcId:          1,
		Attributes:    input.Attributes,
	}).To_Document()

	log.Infof("downloadAndUploadOne - %s → doc %d (%d bytes) getFile=%v upload=%v total=%v",
		input.BotFileUniqueId, documentId, fileSize,
		tGetFile, tUpload, time.Since(start))

	return document, nil
}

// stickerExtForMime returns the file extension for a sticker MIME type.
func stickerExtForMime(mimeType string) string {
	switch mimeType {
	case "application/x-tgsticker":
		return ".tgs"
	case "video/webm":
		return ".webm"
	default:
		return ".webp"
	}
}

// getStorageFileTypeConstructor returns the TL constructor for a file extension.
// Matches the logic in dfs/internal/model/image_util.go.
func getStorageFileTypeConstructor(ext string) int32 {
	switch ext {
	case ".jpeg", ".jpg":
		return int32(mtproto.CRC32_storage_fileJpeg)
	case ".gif":
		return int32(mtproto.CRC32_storage_fileGif)
	case ".png":
		return int32(mtproto.CRC32_storage_filePng)
	case ".pdf":
		return int32(mtproto.CRC32_storage_filePdf)
	case ".mp3":
		return int32(mtproto.CRC32_storage_fileMp3)
	case ".mov":
		return int32(mtproto.CRC32_storage_fileMov)
	case ".mp4":
		return int32(mtproto.CRC32_storage_fileMp4)
	case ".webp":
		return int32(mtproto.CRC32_storage_fileWebp)
	case ".webm":
		return int32(mtproto.CRC32_storage_fileMp4)
	default:
		return int32(mtproto.CRC32_storage_filePartial)
	}
}

// SerializeStickerDoc serializes a Document protobuf to base64 string for DB storage.
func SerializeStickerDoc(doc *mtproto.Document) (string, error) {
	data, err := proto.Marshal(doc)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DeserializeStickerDoc deserializes a base64-encoded Document protobuf from DB.
func DeserializeStickerDoc(s string) (*mtproto.Document, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	doc := &mtproto.Document{}
	return doc, proto.Unmarshal(data, doc)
}

// downloadThumbBytes downloads a thumbnail from Bot API and returns the raw bytes.
func (d *Dao) downloadThumbBytes(ctx context.Context, thumbFileId string) ([]byte, error) {
	// 1. Get file path from Bot API
	fileInfo, err := d.BotAPI.GetFile(ctx, thumbFileId)
	if err != nil {
		return nil, fmt.Errorf("GetFile(thumb): %w", err)
	}

	// 2. Download the thumb bytes (small, ok to buffer)
	data, err := d.BotAPI.DownloadFile(ctx, fileInfo.FilePath)
	if err != nil {
		return nil, fmt.Errorf("DownloadFile(thumb): %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("thumb file is empty")
	}

	return data, nil
}
