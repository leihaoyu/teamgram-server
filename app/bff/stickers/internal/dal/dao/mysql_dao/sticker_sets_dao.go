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

type StickerSetsDAO struct {
	db *sqlx.DB
}

func NewStickerSetsDAO(db *sqlx.DB) *StickerSetsDAO {
	return &StickerSetsDAO{db}
}

// InsertIgnore inserts a sticker set, ignoring duplicate short_name/set_id conflicts.
func (dao *StickerSetsDAO) InsertIgnore(ctx context.Context, do *dataobject.StickerSetsDO) (lastInsertId, rowsAffected int64, err error) {
	var (
		query = "insert ignore into sticker_sets(set_id, access_hash, short_name, title, sticker_type, is_animated, is_video, is_masks, is_emojis, is_official, sticker_count, hash, thumb_doc_id, data_json, fetched_at) values (:set_id, :access_hash, :short_name, :title, :sticker_type, :is_animated, :is_video, :is_masks, :is_emojis, :is_official, :sticker_count, :hash, :thumb_doc_id, :data_json, :fetched_at)"
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

// SelectByShortName
func (dao *StickerSetsDAO) SelectByShortName(ctx context.Context, shortName string) (rValue *dataobject.StickerSetsDO, err error) {
	var (
		query = "select id, set_id, access_hash, short_name, title, sticker_type, is_animated, is_video, is_masks, is_emojis, is_official, sticker_count, hash, thumb_doc_id, data_json, fetched_at from sticker_sets where short_name = ?"
		do    = &dataobject.StickerSetsDO{}
	)
	err = dao.db.QueryRowPartial(ctx, do, query, shortName)

	if err != nil {
		if err != sqlx.ErrNotFound {
			logx.WithContext(ctx).Errorf("queryx in SelectByShortName(_), error: %v", err)
			return
		} else {
			err = nil
		}
	} else {
		rValue = do
	}

	return
}

// SelectBySetId
func (dao *StickerSetsDAO) SelectBySetId(ctx context.Context, setId int64) (rValue *dataobject.StickerSetsDO, err error) {
	var (
		query = "select id, set_id, access_hash, short_name, title, sticker_type, is_animated, is_video, is_masks, is_emojis, is_official, sticker_count, hash, thumb_doc_id, data_json, fetched_at from sticker_sets where set_id = ?"
		do    = &dataobject.StickerSetsDO{}
	)
	err = dao.db.QueryRowPartial(ctx, do, query, setId)

	if err != nil {
		if err != sqlx.ErrNotFound {
			logx.WithContext(ctx).Errorf("queryx in SelectBySetId(_), error: %v", err)
			return
		} else {
			err = nil
		}
	} else {
		rValue = do
	}

	return
}

// SearchByQuery searches sticker sets by title or short_name using LIKE.
func (dao *StickerSetsDAO) SearchByQuery(ctx context.Context, q string, limit int32) (rList []dataobject.StickerSetsDO, err error) {
	var (
		query   = "select id, set_id, access_hash, short_name, title, sticker_type, is_animated, is_video, is_masks, is_emojis, is_official, sticker_count, hash, thumb_doc_id, data_json, fetched_at from sticker_sets where (title like ? or short_name like ?) order by sticker_count desc limit ?"
		pattern = "%" + q + "%"
		values  []dataobject.StickerSetsDO
	)
	err = dao.db.QueryRowsPartial(ctx, &values, query, pattern, pattern, limit)
	if err != nil {
		logx.WithContext(ctx).Errorf("queryx in SearchByQuery(%s), error: %v", q, err)
		return
	}
	rList = values
	return
}

// SelectBySetIds batch-loads sticker set metadata by multiple set_ids.
func (dao *StickerSetsDAO) SelectBySetIds(ctx context.Context, setIds []int64) (rList []dataobject.StickerSetsDO, err error) {
	if len(setIds) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(setIds))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf(
		"select id, set_id, access_hash, short_name, title, sticker_type, is_animated, is_video, is_masks, is_emojis, is_official, sticker_count, hash, thumb_doc_id, data_json, fetched_at from sticker_sets where set_id in (%s)",
		placeholders,
	)
	args := make([]interface{}, 0, len(setIds))
	for _, id := range setIds {
		args = append(args, id)
	}
	var values []dataobject.StickerSetsDO
	err = dao.db.QueryRowsPartial(ctx, &values, query, args...)
	if err != nil {
		logx.WithContext(ctx).Errorf("queryx in SelectBySetIds, error: %v", err)
		return
	}
	rList = values
	return
}
