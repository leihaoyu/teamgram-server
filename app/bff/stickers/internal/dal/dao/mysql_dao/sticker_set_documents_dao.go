package mysql_dao

import (
	"context"
	"database/sql"

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
