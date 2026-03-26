package mysql_dao

import (
	"context"
	"database/sql"

	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dataobject"

	"github.com/zeromicro/go-zero/core/logx"
)

var _ *sql.Result

type UserInstalledStickerSetsDAO struct {
	db *sqlx.DB
}

func NewUserInstalledStickerSetsDAO(db *sqlx.DB) *UserInstalledStickerSetsDAO {
	return &UserInstalledStickerSetsDAO{db}
}

// InsertOrUpdate upserts an installed sticker set for a user.
func (dao *UserInstalledStickerSetsDAO) InsertOrUpdate(ctx context.Context, do *dataobject.UserInstalledStickerSetsDO) (err error) {
	var (
		query = "INSERT INTO user_installed_sticker_sets(user_id, set_id, set_type, order_num, installed_date, archived) VALUES (:user_id, :set_id, :set_type, :order_num, :installed_date, :archived) ON DUPLICATE KEY UPDATE order_num = VALUES(order_num), installed_date = VALUES(installed_date), archived = VALUES(archived), deleted = 0"
	)

	_, err = dao.db.NamedExec(ctx, query, do)
	if err != nil {
		logx.WithContext(ctx).Errorf("namedExec in InsertOrUpdate(%v), error: %v", do, err)
	}

	return
}

// SoftDelete marks a specific installed sticker set as deleted.
func (dao *UserInstalledStickerSetsDAO) SoftDelete(ctx context.Context, userId, setId int64) (rowsAffected int64, err error) {
	var (
		query = "UPDATE user_installed_sticker_sets SET deleted = 1 WHERE user_id = ? AND set_id = ?"
		r     sql.Result
	)

	r, err = dao.db.Exec(ctx, query, userId, setId)
	if err != nil {
		logx.WithContext(ctx).Errorf("exec in SoftDelete(%d, %d), error: %v", userId, setId, err)
		return
	}

	rowsAffected, err = r.RowsAffected()
	if err != nil {
		logx.WithContext(ctx).Errorf("rowsAffected in SoftDelete(%d, %d), error: %v", userId, setId, err)
	}

	return
}

// SelectByUserAndType returns installed sticker sets for a user by type, ordered by order_num.
func (dao *UserInstalledStickerSetsDAO) SelectByUserAndType(ctx context.Context, userId int64, setType int32) (rList []dataobject.UserInstalledStickerSetsDO, err error) {
	var (
		query  = "SELECT id, user_id, set_id, set_type, order_num, installed_date, archived, deleted FROM user_installed_sticker_sets WHERE user_id = ? AND set_type = ? AND deleted = 0 AND archived = 0 ORDER BY order_num ASC"
		values []dataobject.UserInstalledStickerSetsDO
	)
	err = dao.db.QueryRowsPartial(ctx, &values, query, userId, setType)

	if err != nil {
		logx.WithContext(ctx).Errorf("queryx in SelectByUserAndType(%d, %d), error: %v", userId, setType, err)
		return
	}

	rList = values
	return
}

// UpdateOrder updates the order_num for a specific user's sticker set.
func (dao *UserInstalledStickerSetsDAO) UpdateOrder(ctx context.Context, userId, setId int64, orderNum int32) (rowsAffected int64, err error) {
	var (
		query = "UPDATE user_installed_sticker_sets SET order_num = ? WHERE user_id = ? AND set_id = ? AND deleted = 0"
		r     sql.Result
	)

	r, err = dao.db.Exec(ctx, query, orderNum, userId, setId)
	if err != nil {
		logx.WithContext(ctx).Errorf("exec in UpdateOrder(%d, %d, %d), error: %v", userId, setId, orderNum, err)
		return
	}

	rowsAffected, err = r.RowsAffected()
	if err != nil {
		logx.WithContext(ctx).Errorf("rowsAffected in UpdateOrder(%d, %d, %d), error: %v", userId, setId, orderNum, err)
	}

	return
}

// IncrementOrderNum increments order_num for all sets in a given type to make room for a new set at position 0.
func (dao *UserInstalledStickerSetsDAO) IncrementOrderNum(ctx context.Context, userId int64, setType int32) (err error) {
	var (
		query = "UPDATE user_installed_sticker_sets SET order_num = order_num + 1 WHERE user_id = ? AND set_type = ? AND deleted = 0"
	)

	_, err = dao.db.Exec(ctx, query, userId, setType)
	if err != nil {
		logx.WithContext(ctx).Errorf("exec in IncrementOrderNum(%d, %d), error: %v", userId, setType, err)
	}

	return
}

// SelectByUserAndSetId returns a single installed sticker set record for a user.
func (dao *UserInstalledStickerSetsDAO) SelectByUserAndSetId(ctx context.Context, userId, setId int64) (r *dataobject.UserInstalledStickerSetsDO, err error) {
	var (
		query = "SELECT id, user_id, set_id, set_type, order_num, installed_date, archived, deleted FROM user_installed_sticker_sets WHERE user_id = ? AND set_id = ? AND deleted = 0 LIMIT 1"
		value dataobject.UserInstalledStickerSetsDO
	)
	err = dao.db.QueryRowPartial(ctx, &value, query, userId, setId)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return nil, nil
		}
		logx.WithContext(ctx).Errorf("queryx in SelectByUserAndSetId(%d, %d), error: %v", userId, setId, err)
		return
	}
	r = &value
	return
}

// SelectPopularSetIds returns the most popular sticker set_ids ranked by install count.
func (dao *UserInstalledStickerSetsDAO) SelectPopularSetIds(ctx context.Context, limit int32) (rList []int64, err error) {
	return dao.selectPopularSetIdsByType(ctx, 0, limit)
}

// SelectPopularEmojiSetIds returns the most popular emoji sticker set_ids ranked by install count.
func (dao *UserInstalledStickerSetsDAO) SelectPopularEmojiSetIds(ctx context.Context, limit int32) (rList []int64, err error) {
	return dao.selectPopularSetIdsByType(ctx, 2, limit)
}

func (dao *UserInstalledStickerSetsDAO) selectPopularSetIdsByType(ctx context.Context, setType int32, limit int32) (rList []int64, err error) {
	type popularRow struct {
		SetId int64 `db:"set_id"`
	}
	var (
		query  = "SELECT set_id FROM user_installed_sticker_sets WHERE deleted = 0 AND archived = 0 AND set_type = ? GROUP BY set_id ORDER BY COUNT(DISTINCT user_id) DESC LIMIT ?"
		values []popularRow
	)
	err = dao.db.QueryRowsPartial(ctx, &values, query, setType, limit)
	if err != nil {
		logx.WithContext(ctx).Errorf("queryx in selectPopularSetIdsByType(%d), error: %v", setType, err)
		return
	}
	rList = make([]int64, len(values))
	for i, v := range values {
		rList[i] = v.SetId
	}
	return
}
