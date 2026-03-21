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
	"github.com/teamgram/teamgram-server/pkg/imaging"
	"github.com/teamgram/teamgram-server/app/service/dfs/internal/model"
)

// DfsUploadDocumentFileV2
// dfs.uploadDocumentFileV2 creator:long media:InputMedia = Document;
func (c *DfsCore) DfsUploadDocumentFileV2(in *dfs.TLDfsUploadDocumentFileV2) (*mtproto.Document, error) {
	var (
		documentId = c.svcCtx.Dao.IDGenClient2.NextId(c.ctx)
		file       = in.GetMedia().GetFile()
		cacheData  []byte
		mtime      int64
		fileSize   int64
		err        error
	)

	// 有点难理解，主要是为了不在这里引入snowflake
	ext := model.GetFileExtName(file.GetName())
	extType := model.GetStorageFileTypeConstructor(ext)
	accessHash := int64(extType)<<32 | int64(rand.Uint32())

	// Direct mode: file bytes passed inline, skip SSDB entirely
	if len(in.FileData) > 0 {
		cacheData = in.FileData
		mtime = time.Now().Unix()
		fileSize = int64(len(cacheData))
		c.Logger.Infof("dfs.uploadDocumentFile - direct mode: size=%d", fileSize)
	} else {
		r, err2 := c.svcCtx.Dao.OpenFile(c.ctx, in.GetCreator(), file.Id, file.Parts)
		if err2 != nil {
			c.Logger.Errorf("dfs.uploadDocumentFile - %v", err2)
			return nil, mtproto.ErrMediaInvalid
		}

		c.svcCtx.Dao.SetCacheFileInfo(c.ctx, documentId, r.DfsFileInfo)
		mtime = r.DfsFileInfo.Mtime
		fileSize = r.DfsFileInfo.GetFileSize()

		// Pre-read all data from SSDB for MinIO upload
		cacheData, err = r.ReadAll(c.ctx)
		if err != nil {
			c.Logger.Errorf("dfs.uploadDocumentFile - ReadAll failed: %v", err)
			return nil, mtproto.ErrMediaInvalid
		}
	}

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
		Date:          int32(mtime),
		MimeType:      in.GetMedia().GetMimeType(),
		Size2_INT32:   int32(fileSize),
		Size2_INT64:   fileSize,
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

		// cacheData already populated (either direct mode or SSDB ReadAll above)

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

			err = imaging.EncodeJpeg(mThumbData, imaging.FlattenToWhite(mThumb))
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
	if !isThumb {
		var thumbData []byte
		if len(in.ThumbData) > 0 {
			// Direct mode: thumb bytes passed inline
			thumbData = in.ThumbData
		} else if in.GetMedia().GetThumb() != nil {
			// SSDB mode: read thumb from uploaded file parts
			thumbFile := in.GetMedia().GetThumb()
			thumbR, thumbErr := c.svcCtx.Dao.OpenFile(c.ctx, in.GetCreator(), thumbFile.Id, thumbFile.Parts)
			if thumbErr == nil {
				thumbData, _ = thumbR.ReadAll(c.ctx)
			}
		}

		if len(thumbData) > 0 {
			thumbImg, thumbErr3 := imaging.Decode(bytes.NewReader(thumbData))
			if thumbErr3 == nil {
				// Generate stripped size for inline preview
				stripped := bytes2.NewBuffer(make([]byte, 0, 4096))
				if thumbImg.Bounds().Dx() >= thumbImg.Bounds().Dy() {
					thumbErr3 = imaging.EncodeStripped(stripped, imaging.Resize(thumbImg, 40, 0), 30)
				} else {
					thumbErr3 = imaging.EncodeStripped(stripped, imaging.Resize(thumbImg, 0, 40), 30)
				}

				if thumbErr3 == nil {
					// Store original thumbnail data directly (preserve WebP format and transparency)
					path := fmt.Sprintf("%s/%d.dat", mtproto.PhotoSZMediumType, documentId)
					c.svcCtx.Dao.PutPhotoFile(c.ctx, path, thumbData)

					document.Thumbs = []*mtproto.PhotoSize{
						mtproto.MakeTLPhotoStrippedSize(&mtproto.PhotoSize{
							Type:  mtproto.PhotoSZStrippedType,
							Bytes: stripped.Bytes(),
						}).To_PhotoSize(),
						mtproto.MakeTLPhotoSize(&mtproto.PhotoSize{
							Type:  mtproto.PhotoSZMediumType,
							W:     int32(thumbImg.Bounds().Dx()),
							H:     int32(thumbImg.Bounds().Dy()),
							Size2: int32(len(thumbData)),
						}).To_PhotoSize(),
					}
					c.Logger.Infof("dfs.uploadDocumentFile - stored original thumbnail for doc %d (%d bytes)", documentId, len(thumbData))
				}
			} else {
				c.Logger.Infof("dfs.uploadDocumentFile - cannot decode external thumbnail: %v", thumbErr3)
			}
		}
	}

	// Write document file to MinIO
	// cacheData is always pre-populated at this point
	minioPath := fmt.Sprintf("%d.dat", documentId)
	startTime := time.Now()

	uploadInfo, err2 := c.svcCtx.Dao.PutDocumentFile(c.ctx,
		minioPath,
		bytes.NewReader(cacheData))
	elapsed := time.Since(startTime)
	if err2 != nil {
		c.Logger.Errorf("dfs.uploadDocumentFile - minio put FAILED: path=%s, size=%d, elapsed=%v, error=%v",
			minioPath, len(cacheData), elapsed, err2)
		return nil, fmt.Errorf("minio put failed for %s: %w", minioPath, err2)
	}
	c.Logger.Infof("dfs.uploadDocumentFile - minio put OK: path=%s, inputSize=%d, uploadedSize=%d, bucket=%s, elapsed=%v",
		minioPath, len(cacheData), uploadInfo.Size, uploadInfo.Bucket, elapsed)
	if uploadInfo.Size != int64(len(cacheData)) {
		c.Logger.Errorf("dfs.uploadDocumentFile - minio SIZE MISMATCH: path=%s, expected=%d, actual=%d",
			minioPath, len(cacheData), uploadInfo.Size)
		return nil, fmt.Errorf("minio size mismatch for %s: expected=%d, actual=%d", minioPath, len(cacheData), uploadInfo.Size)
	}

	return document, nil
}
