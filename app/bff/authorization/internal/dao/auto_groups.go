package dao

import (
	"context"

	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/zeromicro/go-zero/core/logx"
)

// GetCurrentAutoGroup returns the current active (non-full) auto group for the given type and key.
// Must be called within a transaction with FOR UPDATE for concurrency safety.
func (d *Dao) GetCurrentAutoGroup(ctx context.Context, groupType int32, groupKey string) (*AutoGroupDO, error) {
	if d.AutoGroupDB == nil {
		return nil, nil
	}

	var do AutoGroupDO
	err := d.AutoGroupDB.QueryRowPartial(ctx, &do,
		"SELECT id, group_type, group_key, sequence_num, chat_id, creator_user_id, participant_count, is_full "+
			"FROM auto_groups WHERE group_type = ? AND group_key = ? AND is_full = 0 "+
			"ORDER BY sequence_num DESC LIMIT 1",
		groupType, groupKey)

	if err != nil {
		if err == sqlx.ErrNotFound {
			return nil, nil
		}
		logx.WithContext(ctx).Errorf("GetCurrentAutoGroup error: %v", err)
		return nil, err
	}
	return &do, nil
}

// GetCurrentAutoGroupTx is the transaction version of GetCurrentAutoGroup, with FOR UPDATE lock.
func (d *Dao) GetCurrentAutoGroupTx(tx *sqlx.Tx, groupType int32, groupKey string) (*AutoGroupDO, error) {
	var do AutoGroupDO
	err := tx.QueryRowPartial(&do,
		"SELECT id, group_type, group_key, sequence_num, chat_id, creator_user_id, participant_count, is_full "+
			"FROM auto_groups WHERE group_type = ? AND group_key = ? AND is_full = 0 "+
			"ORDER BY sequence_num DESC LIMIT 1 FOR UPDATE",
		groupType, groupKey)

	if err != nil {
		if err == sqlx.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &do, nil
}

// GetMaxSequenceNum returns the latest sequence number for the given group type and key.
func (d *Dao) GetMaxSequenceNumTx(tx *sqlx.Tx, groupType int32, groupKey string) (int32, error) {
	var seq int32
	err := tx.QueryRowPartial(&seq,
		"SELECT COALESCE(MAX(sequence_num), 0) FROM auto_groups WHERE group_type = ? AND group_key = ?",
		groupType, groupKey)
	if err != nil {
		return 0, err
	}
	return seq, nil
}

// CreateAutoGroup inserts a new auto group record.
func (d *Dao) CreateAutoGroupTx(tx *sqlx.Tx, do *AutoGroupDO) error {
	_, err := tx.Exec(
		"INSERT INTO auto_groups (group_type, group_key, sequence_num, chat_id, creator_user_id, participant_count, is_full) "+
			"VALUES (?, ?, ?, ?, ?, ?, 0)",
		do.GroupType, do.GroupKey, do.SequenceNum, do.ChatId, do.CreatorUserId, do.ParticipantCount)
	return err
}

// IncrParticipantCountTx atomically increments the participant count and returns the new value.
func (d *Dao) IncrParticipantCountTx(tx *sqlx.Tx, chatId int64) (int32, error) {
	_, err := tx.Exec(
		"UPDATE auto_groups SET participant_count = participant_count + 1 WHERE chat_id = ?",
		chatId)
	if err != nil {
		return 0, err
	}

	var count int32
	err = tx.QueryRowPartial(&count,
		"SELECT participant_count FROM auto_groups WHERE chat_id = ?",
		chatId)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// MarkAutoGroupFullTx marks the auto group as full.
func (d *Dao) MarkAutoGroupFullTx(tx *sqlx.Tx, chatId int64) error {
	_, err := tx.Exec(
		"UPDATE auto_groups SET is_full = 1 WHERE chat_id = ?",
		chatId)
	return err
}
