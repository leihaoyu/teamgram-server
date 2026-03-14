package dao

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"path"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/service/dfs/dfs"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	filePartSize    = 512 * 1024 // 512KB per part
	downloadWorkers = 3
)

// StickerDownloadInput holds the info needed to download one sticker file and upload it to DFS.
type StickerDownloadInput struct {
	BotFileId       string
	BotFileUniqueId string
	MimeType        string
	Attributes      []*mtproto.DocumentAttribute
}

// DownloadAndUploadStickerFiles downloads sticker files from Telegram Bot API and uploads them
// to DFS (MinIO) synchronously. Returns DFS-backed Documents in the same order as inputs.
// If any file fails, returns an error (caller should not cache partial results).
func (d *Dao) DownloadAndUploadStickerFiles(ctx context.Context, inputs []StickerDownloadInput) ([]*mtproto.Document, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	results := make([]*mtproto.Document, len(inputs))
	var (
		mu      sync.Mutex
		firstErr error
	)

	sem := make(chan struct{}, downloadWorkers)
	var wg sync.WaitGroup

	for i := range inputs {
		idx := i
		input := inputs[i]

		wg.Add(1)
		sem <- struct{}{}
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
		return nil, firstErr
	}

	return results, nil
}

// downloadAndUploadOne downloads a single sticker file from Bot API and uploads it to DFS.
// Returns the DFS-backed Document with the real DFS-assigned ID.
func (d *Dao) downloadAndUploadOne(ctx context.Context, input *StickerDownloadInput) (*mtproto.Document, error) {
	log := logx.WithContext(ctx)

	// 1. Get file path from Bot API
	fileInfo, err := d.BotAPI.GetFile(ctx, input.BotFileId)
	if err != nil {
		return nil, fmt.Errorf("GetFile: %w", err)
	}

	// 2. Download the file bytes
	data, err := d.BotAPI.DownloadFile(ctx, fileInfo.FilePath)
	if err != nil {
		return nil, fmt.Errorf("DownloadFile: %w", err)
	}

	// 3. Use a temporary fileId for SSDB parts (IDGen gives us a unique key)
	tempFileId := d.IDGenClient2.NextId(ctx)

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

		_, err = d.DfsClient.DfsWriteFilePartData(ctx, &dfs.TLDfsWriteFilePartData{
			Creator:        tempFileId,
			FileId:         tempFileId,
			FilePart:       part,
			Bytes:          data[start:end],
			Big:            false,
			FileTotalParts: &types.Int32Value{Value: totalParts},
		})
		if err != nil {
			return nil, fmt.Errorf("DfsWriteFilePartData(part=%d): %w", part, err)
		}
	}

	// 4. Finalize to MinIO via DfsUploadDocumentFileV2
	ext := path.Ext(fileInfo.FilePath)
	if ext == "" {
		ext = ".dat"
	}

	dfsDoc, err := d.DfsClient.DfsUploadDocumentFileV2(ctx, &dfs.TLDfsUploadDocumentFileV2{
		Creator: tempFileId,
		Media: mtproto.MakeTLInputMediaUploadedDocument(&mtproto.InputMedia{
			File: mtproto.MakeTLInputFile(&mtproto.InputFile{
				Id:    tempFileId,
				Parts: totalParts,
				Name:  input.BotFileUniqueId + ext,
			}).To_InputFile(),
			MimeType:   input.MimeType,
			Attributes: input.Attributes,
		}).To_InputMedia(),
	})
	if err != nil {
		return nil, fmt.Errorf("DfsUploadDocumentFileV2: %w", err)
	}

	log.Infof("downloadAndUploadOne - %s → dfs %d (%d bytes, %d parts)",
		input.BotFileUniqueId, dfsDoc.GetId(), len(data), totalParts)

	return dfsDoc, nil
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
