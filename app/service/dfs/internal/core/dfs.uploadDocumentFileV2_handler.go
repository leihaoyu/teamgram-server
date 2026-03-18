/*
 * Created from 'scheme.tl' by 'mtprotoc'
 *
 * Copyright (c) 2021-present,  Teamgram Studio (https://teamgram.io).
 *  All rights reserved.
 *
 * Author: teamgramio (teamgram.io@gmail.com)
 */

package core

import (
	"bytes"
	"fmt"
	"image"
	"math/rand"
	"time"

	"github.com/teamgram/marmota/pkg/bytes2"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/service/dfs/dfs"
	"github.com/teamgram/teamgram-server/app/service/dfs/internal/imaging"
	"github.com/teamgram/teamgram-server/app/service/dfs/internal/model"
)

// DfsUploadDocumentFileV2
// dfs.uploadDocumentFileV2 creator:long media:InputMedia = Document;
func (c *DfsCore) DfsUploadDocumentFileV2(in *dfs.TLDfsUploadDocumentFileV2) (*mtproto.Document, error) {
	var (
		documentId = c.svcCtx.Dao.IDGenClient2.NextId(c.ctx)
		//idgen.GetUUID()
		file      = in.GetMedia().GetFile()
		cacheData []byte
	)

	// 有点难理解，主要是为了不在这里引入snowflake
	ext := model.GetFileExtName(file.GetName())
	extType := model.GetStorageFileTypeConstructor(ext)
	accessHash := int64(extType)<<32 | int64(rand.Uint32())

	r, err := c.svcCtx.Dao.OpenFile(c.ctx, in.GetCreator(), file.Id, file.Parts)
	if err != nil {
		c.Logger.Errorf("dfs.uploadDocumentFile - %v", err)
		return nil, mtproto.ErrMediaInvalid
	}

	//fileInfo, err := s.Dao.GetFileInfo(ctx, creatorId, file.Id)
	//if err != nil {
	//	log.Errorf("dfs.uploadDocumentFile - error: %v", err)
	//	return nil, err
	//}

	c.svcCtx.Dao.SetCacheFileInfo(c.ctx, documentId, r.DfsFileInfo)

	//go func() {
	//	_, err2 := s.Dao.PutDocumentFile(ctx,
	//		fmt.Sprintf("%d.dat", documentId),
	//		s.Dao.NewSSDBReader(r.DfsFileInfo))
	//	if err2 != nil {
	//		log.Errorf("dfs.uploadDocumentFile - error: %v", err2)
	//	}
	//}()

	attributes := make([]*mtproto.DocumentAttribute, 0, len(in.GetMedia().Attributes))
	for _, attr := range in.GetMedia().GetAttributes() {
		switch attr.GetPredicateName() {
		case mtproto.Predicate_documentAttributeAnimated:
		case mtproto.Predicate_documentAttributeFilename:
			if attr.GetFileName() != "" {
				attributes = append(attributes, attr)
			}
		case mtproto.Predicate_documentAttributeAudio:
			if in.GetMedia().GetMimeType() == "audio/ogg" {
				if attr.Voice == true {
					attributes = append(attributes, attr)
				}
			} else {
				attributes = append(attributes, attr)
			}
		default:
			attributes = append(attributes, attr)
		}
	}

	// document#1e87342b flags:#
	//	id:long
	//	access_hash:long
	//	file_reference:bytes
	//	date:int
	//	mime_type:string
	//	size:int
	//	thumbs:flags.0?Vector<PhotoSize>
	//	video_thumbs:flags.1?Vector<VideoSize>
	//	dc_id:int
	//	attributes:Vector<DocumentAttribute> = Document;
	document := mtproto.MakeTLDocument(&mtproto.Document{
		Id:            documentId,
		AccessHash:    accessHash,
		FileReference: []byte{},
		Date:          int32(r.DfsFileInfo.Mtime),
		MimeType:      in.GetMedia().GetMimeType(),
		Size2_INT32:   int32(r.DfsFileInfo.GetFileSize()),
		Size2_INT64:   r.DfsFileInfo.GetFileSize(),
		Thumbs:        nil,
		VideoThumbs:   nil,
		DcId:          1,
		Attributes:    attributes,
	}).To_Document()

	isThumb := mtproto.IsMimeAcceptedForPhotoVideoAlbum(document.MimeType) && model.IsFileExtImage(ext)
	if isThumb {
		var (
			thumb image.Image
		)

		cacheData, err = r.ReadAll(c.ctx)
		if err != nil {
			c.Logger.Errorf("dfs.uploadDocumentFile - %v", err)
			return nil, mtproto.ErrWallpaperFileInvalid
		}

		// build photoStrippedSize
		thumb, err = imaging.Decode(bytes.NewReader(cacheData))
		if err == nil {
			stripped := bytes2.NewBuffer(make([]byte, 0, 4096))
			if thumb.Bounds().Dx() >= thumb.Bounds().Dy() {
				err = imaging.EncodeStripped(stripped, imaging.Resize(thumb, 40, 0), 30)
			} else {
				err = imaging.EncodeStripped(stripped, imaging.Resize(thumb, 0, 40), 30)
			}
			if err != nil {
				c.Logger.Errorf("dfs.uploadDocumentFile - error: %v", err)
				return nil, err
			}

			// upload thumb
			var (
				mThumbData = bytes2.NewBuffer(make([]byte, 0, len(cacheData)))
				mThumb     image.Image
			)
			if thumb.Bounds().Dx() >= thumb.Bounds().Dy() {
				mThumb = imaging.Resize(thumb, 320, 0)
			} else {
				mThumb = imaging.Resize(thumb, 0, 320)
			}

			err = imaging.EncodeJpeg(mThumbData, mThumb)
			if err != nil {
				c.Logger.Errorf("dfs.uploadDocumentFile - error: %v", err)
				return nil, err
			}

			// upload thumb
			path := fmt.Sprintf("%s/%d.dat", mtproto.PhotoSZMediumType, documentId)
			c.svcCtx.Dao.PutPhotoFile(c.ctx, path, mThumbData.Bytes())

			document.Thumbs = []*mtproto.PhotoSize{
				mtproto.MakeTLPhotoStrippedSize(&mtproto.PhotoSize{
					Type:  mtproto.PhotoSZStrippedType,
					Bytes: stripped.Bytes(),
				}).To_PhotoSize(),
				mtproto.MakeTLPhotoSize(&mtproto.PhotoSize{
					Type:  mtproto.PhotoSZMediumType,
					W:     int32(mThumb.Bounds().Dx()),
					H:     int32(mThumb.Bounds().Dy()),
					Size2: int32(len(mThumbData.Bytes())),
				}).To_PhotoSize(),
			}
		} else {
			c.Logger.Errorf("dfs.uploadDocumentFile - error: %v", err)
			isThumb = false
		}
	}

	// Handle externally-provided thumbnail (e.g. sticker thumbnails from Bot API)
	if !isThumb && in.GetMedia().GetThumb() != nil {
		thumbFile := in.GetMedia().GetThumb()
		thumbR, thumbErr := c.svcCtx.Dao.OpenFile(c.ctx, in.GetCreator(), thumbFile.Id, thumbFile.Parts)
		if thumbErr == nil {
			thumbData, thumbErr2 := thumbR.ReadAll(c.ctx)
			if thumbErr2 == nil && len(thumbData) > 0 {
				thumbImg, thumbErr3 := imaging.Decode(bytes.NewReader(thumbData))
				if thumbErr3 == nil {
					// Generate stripped size
					stripped := bytes2.NewBuffer(make([]byte, 0, 4096))
					if thumbImg.Bounds().Dx() >= thumbImg.Bounds().Dy() {
						thumbErr3 = imaging.EncodeStripped(stripped, imaging.Resize(thumbImg, 40, 0), 30)
					} else {
						thumbErr3 = imaging.EncodeStripped(stripped, imaging.Resize(thumbImg, 0, 40), 30)
					}

					if thumbErr3 == nil {
						// Encode thumb as JPEG for storage
						mThumbData := bytes2.NewBuffer(make([]byte, 0, len(thumbData)))
						var mThumb image.Image
						if thumbImg.Bounds().Dx() >= thumbImg.Bounds().Dy() {
							mThumb = imaging.Resize(thumbImg, 128, 0)
						} else {
							mThumb = imaging.Resize(thumbImg, 0, 128)
						}

						if encErr := imaging.EncodeJpeg(mThumbData, mThumb); encErr == nil {
							path := fmt.Sprintf("%s/%d.dat", mtproto.PhotoSZMediumType, documentId)
							c.svcCtx.Dao.PutPhotoFile(c.ctx, path, mThumbData.Bytes())

							document.Thumbs = []*mtproto.PhotoSize{
								mtproto.MakeTLPhotoStrippedSize(&mtproto.PhotoSize{
									Type:  mtproto.PhotoSZStrippedType,
									Bytes: stripped.Bytes(),
								}).To_PhotoSize(),
								mtproto.MakeTLPhotoSize(&mtproto.PhotoSize{
									Type:  mtproto.PhotoSZMediumType,
									W:     int32(mThumb.Bounds().Dx()),
									H:     int32(mThumb.Bounds().Dy()),
									Size2: int32(len(mThumbData.Bytes())),
								}).To_PhotoSize(),
							}
							isThumb = true
							c.Logger.Infof("dfs.uploadDocumentFile - generated thumb from external thumbnail for doc %d", documentId)
						}
					}
				} else {
					c.Logger.Infof("dfs.uploadDocumentFile - cannot decode external thumbnail: %v", thumbErr3)
				}
			}
		}
	}

	// Sync write to MinIO to ensure file is persisted before SSDB cache expires
	minioPath := fmt.Sprintf("%d.dat", documentId)
	startTime := time.Now()

	if isThumb {
		uploadInfo, err2 := c.svcCtx.Dao.PutDocumentFile(c.ctx,
			minioPath,
			bytes.NewReader(cacheData))
		elapsed := time.Since(startTime)
		if err2 != nil {
			c.Logger.Errorf("dfs.uploadDocumentFile - minio put FAILED: path=%s, size=%d, elapsed=%v, error=%v",
				minioPath, len(cacheData), elapsed, err2)
		} else {
			c.Logger.Infof("dfs.uploadDocumentFile - minio put OK: path=%s, inputSize=%d, uploadedSize=%d, bucket=%s, elapsed=%v",
				minioPath, len(cacheData), uploadInfo.Size, uploadInfo.Bucket, elapsed)
		}
	} else {
		fileSize := r.DfsFileInfo.GetFileSize()
		uploadInfo, err2 := c.svcCtx.Dao.PutDocumentFile(c.ctx,
			minioPath,
			c.svcCtx.Dao.NewSSDBReader(r.DfsFileInfo))
		elapsed := time.Since(startTime)
		if err2 != nil {
			c.Logger.Errorf("dfs.uploadDocumentFile - minio put FAILED: path=%s, expectedSize=%d, elapsed=%v, error=%v",
				minioPath, fileSize, elapsed, err2)
		} else {
			c.Logger.Infof("dfs.uploadDocumentFile - minio put OK: path=%s, expectedSize=%d, uploadedSize=%d, bucket=%s, elapsed=%v",
				minioPath, fileSize, uploadInfo.Size, uploadInfo.Bucket, elapsed)
			if uploadInfo.Size != fileSize {
				c.Logger.Errorf("dfs.uploadDocumentFile - minio SIZE MISMATCH: path=%s, expected=%d, actual=%d",
					minioPath, fileSize, uploadInfo.Size)
			}
		}
	}

	return document, nil
}
