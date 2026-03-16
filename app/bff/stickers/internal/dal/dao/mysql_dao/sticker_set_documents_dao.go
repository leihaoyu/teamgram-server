package mysql_dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"

	"github.com/zeromicro/go-zero/core/logx"
)

var _ *sql.Result

type StickerSetDocumentsDAO struct {
	db *sqlx.DB
}

func NewStickerSetDocumentsDAO(db *sqlx.DB) *StickerSetDocumentsDAO {
	return &StickerSetDocumentsDAO{db}
}

// InsertIgnore inserts a sticker document, ignoring duplicate document_id conflicts.
func (dao *StickerSetDocumentsDAO) InsertIgnore(ctx context.Context, do *dataobject.StickerSetDocumentsDO) (lastInsertId, rowsAffected int64, err error) {
	var (
		query = "insert ignore into sticker_set_documents(set_id, document_id, sticker_index, emoji, bot_file_id, bot_file_unique_id, bot_thumb_file_id, document_data, file_downloaded) values (:set_id, :document_id, :sticker_index, :emoji, :bot_file_id, :bot_file_unique_id, :bot_thumb_file_id, :document_data, :file_downloaded)"
		r     sql.Result
	)

	r, err = dao.db.NamedExec(ctx, query, do)
	if err != nil {
		logx.WithContext(ctx).Errorf("namedExec in InsertIgnore(%v), error: %v", do, err)
		return
	}

	lastInsertId, err = r.LastInsertId()
	if err != nil {
		logx.WithContext(ctx).Errorf("lastInsertId in InsertIgnore(%v)_error: %v", do, err)
		return
	}
	rowsAffected, err = r.RowsAffected()
	if err != nil {
		logx.WithContext(ctx).Errorf("rowsAffected in InsertIgnore(%v)_error: %v", do, err)
	}

	return
}

// SelectBySetId
func (dao *StickerSetDocumentsDAO) SelectBySetId(ctx context.Context, setId int64) (rList []dataobject.StickerSetDocumentsDO, err error) {
	var (
		query  = "select id, set_id, document_id, sticker_index, emoji, bot_file_id, bot_file_unique_id, bot_thumb_file_id, document_data, file_downloaded from sticker_set_documents where set_id = ? order by sticker_index asc"
		values []dataobject.StickerSetDocumentsDO
	)
	err = dao.db.QueryRowsPartial(ctx, &values, query, setId)

	if err != nil {
		logx.WithContext(ctx).Errorf("queryx in SelectBySetId(_), error: %v", err)
		return
	}

	rList = values
	return
}

// SelectPendingDownloadBySetId
func (dao *StickerSetDocumentsDAO) SelectPendingDownloadBySetId(ctx context.Context, setId int64) (rList []dataobject.StickerSetDocumentsDO, err error) {
	var (
		query  = "select id, set_id, document_id, sticker_index, emoji, bot_file_id, bot_file_unique_id, bot_thumb_file_id, document_data, file_downloaded from sticker_set_documents where set_id = ? and file_downloaded = 0 order by sticker_index asc"
		values []dataobject.StickerSetDocumentsDO
	)
	err = dao.db.QueryRowsPartial(ctx, &values, query, setId)

	if err != nil {
		logx.WithContext(ctx).Errorf("queryx in SelectPendingDownloadBySetId(_), error: %v", err)
		return
	}

	rList = values
	return
}

// UpdateDocumentAfterDFSUpload replaces oldDocumentId with newDocumentId, updates document_data,
// and sets file_downloaded = 1, after DfsUploadDocumentFileV2 assigns a real DFS-backed ID.
func (dao *StickerSetDocumentsDAO) UpdateDocumentAfterDFSUpload(ctx context.Context, oldDocumentId, newDocumentId int64, newDocumentData string) (rowsAffected int64, err error) {
	var (
		query = "update sticker_set_documents set document_id = ?, document_data = ?, file_downloaded = 1 where document_id = ?"
		r     sql.Result
	)

	r, err = dao.db.Exec(ctx, query, newDocumentId, newDocumentData, oldDocumentId)
	if err != nil {
		logx.WithContext(ctx).Errorf("exec in UpdateDocumentAfterDFSUpload(_), error: %v", err)
		return
	}

	rowsAffected, err = r.RowsAffected()
	if err != nil {
		logx.WithContext(ctx).Errorf("rowsAffected in UpdateDocumentAfterDFSUpload(_), error: %v", err)
	}

	return
}

// SelectBySetIdsAndEmoji returns sticker documents matching a specific emoji across multiple set_ids.
func (dao *StickerSetDocumentsDAO) SelectBySetIdsAndEmoji(ctx context.Context, setIds []int64, emoji string) (rList []dataobject.StickerSetDocumentsDO, err error) {
	if len(setIds) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(setIds))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf(
		"select id, set_id, document_id, sticker_index, emoji, bot_file_id, bot_file_unique_id, bot_thumb_file_id, document_data, file_downloaded from sticker_set_documents where set_id in (%s) and emoji = ? order by set_id, sticker_index asc",
		placeholders,
	)
	args := make([]interface{}, 0, len(setIds)+1)
	for _, id := range setIds {
		args = append(args, id)
	}
	args = append(args, emoji)

	var values []dataobject.StickerSetDocumentsDO
	err = dao.db.QueryRowsPartial(ctx, &values, query, args...)
	if err != nil {
		logx.WithContext(ctx).Errorf("queryx in SelectBySetIdsAndEmoji, error: %v", err)
		return
	}
	rList = values
	return
}

// SelectFirstBySetId returns the first (cover) document from a sticker set.
func (dao *StickerSetDocumentsDAO) SelectFirstBySetId(ctx context.Context, setId int64) (rValue *dataobject.StickerSetDocumentsDO, err error) {
	var (
		query = "select id, set_id, document_id, sticker_index, emoji, bot_file_id, bot_file_unique_id, bot_thumb_file_id, document_data, file_downloaded from sticker_set_documents where set_id = ? order by sticker_index asc limit 1"
		do    = &dataobject.StickerSetDocumentsDO{}
	)
	err = dao.db.QueryRowPartial(ctx, do, query, setId)
	if err != nil {
		if err != sqlx.ErrNotFound {
			logx.WithContext(ctx).Errorf("queryx in SelectFirstBySetId(%d), error: %v", setId, err)
			return
		}
		err = nil
	} else {
		rValue = do
	}
	return
}

// UpdateFileDownloaded
func (dao *StickerSetDocumentsDAO) UpdateFileDownloaded(ctx context.Context, documentId int64) (rowsAffected int64, err error) {
	var (
		query = "update sticker_set_documents set file_downloaded = 1 where document_id = ?"
		r     sql.Result
	)

	r, err = dao.db.Exec(ctx, query, documentId)
	if err != nil {
		logx.WithContext(ctx).Errorf("exec in UpdateFileDownloaded(_), error: %v", err)
		return
	}

	rowsAffected, err = r.RowsAffected()
	if err != nil {
		logx.WithContext(ctx).Errorf("rowsAffected in UpdateFileDownloaded(_), error: %v", err)
	}

	return
}
