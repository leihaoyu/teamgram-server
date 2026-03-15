package mysql_dao

import (
	"context"
	"database/sql"

	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"

	"github.com/zeromicro/go-zero/core/logx"
)

var _ *sql.Result

type UserFavedStickersDAO struct {
	db *sqlx.DB
}

func NewUserFavedStickersDAO(db *sqlx.DB) *UserFavedStickersDAO {
	return &UserFavedStickersDAO{db}
}

// InsertOrUpdate upserts a faved sticker for a user.
func (dao *UserFavedStickersDAO) InsertOrUpdate(ctx context.Context, do *dataobject.UserFavedStickersDO) (err error) {
	var (
		query = "INSERT INTO user_faved_stickers(user_id, document_id, emoji, document_data, date2) VALUES (:user_id, :document_id, :emoji, :document_data, :date2) ON DUPLICATE KEY UPDATE document_data = VALUES(document_data), emoji = VALUES(emoji), date2 = VALUES(date2), deleted = 0"
	)

	_, err = dao.db.NamedExec(ctx, query, do)
	if err != nil {
		logx.WithContext(ctx).Errorf("namedExec in InsertOrUpdate(%v), error: %v", do, err)
	}

	return
}

// SoftDelete marks a specific faved sticker as deleted.
func (dao *UserFavedStickersDAO) SoftDelete(ctx context.Context, userId, documentId int64) (rowsAffected int64, err error) {
	var (
		query = "UPDATE user_faved_stickers SET deleted = 1 WHERE user_id = ? AND document_id = ?"
		r     sql.Result
	)

	r, err = dao.db.Exec(ctx, query, userId, documentId)
	if err != nil {
		logx.WithContext(ctx).Errorf("exec in SoftDelete(%d, %d), error: %v", userId, documentId, err)
		return
	}

	rowsAffected, err = r.RowsAffected()
	if err != nil {
		logx.WithContext(ctx).Errorf("rowsAffected in SoftDelete(%d, %d), error: %v", userId, documentId, err)
	}

	return
}

// SelectByUser returns faved stickers for a user, ordered by most recent first.
func (dao *UserFavedStickersDAO) SelectByUser(ctx context.Context, userId int64, limit int) (rList []dataobject.UserFavedStickersDO, err error) {
	var (
		query  = "SELECT id, user_id, document_id, emoji, document_data, date2, deleted FROM user_faved_stickers WHERE user_id = ? AND deleted = 0 ORDER BY date2 DESC LIMIT ?"
		values []dataobject.UserFavedStickersDO
	)
	err = dao.db.QueryRowsPartial(ctx, &values, query, userId, limit)

	if err != nil {
		logx.WithContext(ctx).Errorf("queryx in SelectByUser(%d, %d), error: %v", userId, limit, err)
		return
	}

	rList = values
	return
}
