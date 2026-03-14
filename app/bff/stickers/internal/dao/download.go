package dao

import (
	"context"
	"math"

	"github.com/teamgram/teamgram-server/app/service/dfs/dfs"

	"github.com/gogo/protobuf/types"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	filePartSize    = 512 * 1024 // 512KB per part
	downloadWorkers = 3
)

// DownloadStickerFiles downloads all pending sticker files for a set from Telegram Bot API
// and stores them via DFS service.
func (d *Dao) DownloadStickerFiles(ctx context.Context, setId int64) {
	log := logx.WithContext(ctx)

	docs, err := d.StickerSetDocumentsDAO.SelectPendingDownloadBySetId(ctx, setId)
	if err != nil {
		log.Errorf("DownloadStickerFiles - SelectPendingDownload error: %v", err)
		return
	}

	if len(docs) == 0 {
		return
	}

	// Use a semaphore channel to limit concurrency
	sem := make(chan struct{}, downloadWorkers)

	for i := range docs {
		doc := docs[i]
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					logx.WithContext(ctx).Errorf("downloadOneStickerFile panic for doc %d: %v", doc.DocumentId, r)
				}
			}()
			d.downloadOneStickerFile(ctx, doc.DocumentId, doc.BotFileId)
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < downloadWorkers; i++ {
		sem <- struct{}{}
	}

	log.Infof("DownloadStickerFiles - finished downloading %d files for set %d", len(docs), setId)
}

func (d *Dao) downloadOneStickerFile(ctx context.Context, documentId int64, botFileId string) {
	log := logx.WithContext(ctx)

	// 1. Get file path from Bot API
	fileInfo, err := d.BotAPI.GetFile(ctx, botFileId)
	if err != nil {
		log.Errorf("downloadOneStickerFile - GetFile(%s) error: %v", botFileId, err)
		return
	}

	// 2. Download the file bytes
	data, err := d.BotAPI.DownloadFile(ctx, fileInfo.FilePath)
	if err != nil {
		log.Errorf("downloadOneStickerFile - DownloadFile(%s) error: %v", fileInfo.FilePath, err)
		return
	}

	// 3. Write to DFS via file parts
	// Use documentId as both creator and fileId for DFS
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
		partData := data[start:end]

		_, err = d.DfsClient.DfsWriteFilePartData(ctx, &dfs.TLDfsWriteFilePartData{
			Creator:        documentId,
			FileId:         documentId,
			FilePart:       part,
			Bytes:          partData,
			Big:            false,
			FileTotalParts: &types.Int32Value{Value: totalParts},
		})
		if err != nil {
			log.Errorf("downloadOneStickerFile - DfsWriteFilePartData(doc=%d, part=%d) error: %v",
				documentId, part, err)
			return
		}
	}

	// 4. Mark as downloaded
	_, err = d.StickerSetDocumentsDAO.UpdateFileDownloaded(ctx, documentId)
	if err != nil {
		log.Errorf("downloadOneStickerFile - UpdateFileDownloaded(%d) error: %v", documentId, err)
		return
	}

	log.Infof("downloadOneStickerFile - successfully downloaded doc %d (%d bytes, %d parts)",
		documentId, len(data), totalParts)
}
