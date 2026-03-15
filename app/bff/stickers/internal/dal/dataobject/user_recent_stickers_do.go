package dataobject

type UserRecentStickersDO struct {
	Id           int64  `db:"id"`
	UserId       int64  `db:"user_id"`
	DocumentId   int64  `db:"document_id"`
	Emoji        string `db:"emoji"`
	DocumentData string `db:"document_data"`
	Date2        int64  `db:"date2"`
	Deleted      bool   `db:"deleted"`
}
