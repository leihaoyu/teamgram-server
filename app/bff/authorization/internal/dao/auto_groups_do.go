package dao

type AutoGroupDO struct {
	Id               int64  `db:"id"`
	GroupType        int32  `db:"group_type"`
	GroupKey         string `db:"group_key"`
	SequenceNum      int32  `db:"sequence_num"`
	ChatId           int64  `db:"chat_id"`
	CreatorUserId    int64  `db:"creator_user_id"`
	ParticipantCount int32  `db:"participant_count"`
	IsFull           int32  `db:"is_full"`
}

const (
	AutoGroupTypeGeneral = 1
	AutoGroupTypeCity    = 2
)
