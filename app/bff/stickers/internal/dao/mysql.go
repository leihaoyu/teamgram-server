package dao

import (
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/teamgram-server/app/bff/stickers/internal/dal/dao/mysql_dao"
)

type Mysql struct {
	*sqlx.DB
	*mysql_dao.StickerSetsDAO
	*mysql_dao.StickerSetDocumentsDAO
	*mysql_dao.UserRecentStickersDAO
	*mysql_dao.UserFavedStickersDAO
}

func newMysqlDao(db *sqlx.DB) *Mysql {
	return &Mysql{
		DB:                     db,
		StickerSetsDAO:         mysql_dao.NewStickerSetsDAO(db),
		StickerSetDocumentsDAO: mysql_dao.NewStickerSetDocumentsDAO(db),
		UserRecentStickersDAO:  mysql_dao.NewUserRecentStickersDAO(db),
		UserFavedStickersDAO:   mysql_dao.NewUserFavedStickersDAO(db),
	}
}
