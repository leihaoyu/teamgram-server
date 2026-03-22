package dao

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

type DevicesDO struct {
	AuthKeyId    int64
	UserId       int64
	TokenType    int32
	Token        string
	NoMuted      bool
	AppSandbox   bool
	Secret       string
	OtherUids    string
}

func (d *Dao) RegisterDevice(ctx context.Context, do *DevicesDO) error {
	if d.DevicesDB == nil {
		logx.WithContext(ctx).Errorf("RegisterDevice: DevicesDB is nil")
		return fmt.Errorf("devices database not configured")
	}

	query := "INSERT INTO devices(auth_key_id, user_id, token_type, token, no_muted, app_sandbox, secret, other_uids) " +
		"VALUES (?, ?, ?, ?, ?, ?, ?, ?) " +
		"ON DUPLICATE KEY UPDATE token = VALUES(token), no_muted = VALUES(no_muted), secret = VALUES(secret), other_uids = VALUES(other_uids), state = 0"

	_, err := d.DevicesDB.Exec(ctx, query,
		do.AuthKeyId,
		do.UserId,
		do.TokenType,
		do.Token,
		do.NoMuted,
		do.AppSandbox,
		do.Secret,
		do.OtherUids,
	)
	if err != nil {
		logx.WithContext(ctx).Errorf("RegisterDevice error: %v", err)
	}
	return err
}

func (d *Dao) UnregisterDevice(ctx context.Context, tokenType int32, token string) error {
	if d.DevicesDB == nil {
		logx.WithContext(ctx).Errorf("UnregisterDevice: DevicesDB is nil")
		return fmt.Errorf("devices database not configured")
	}

	query := "UPDATE devices SET state = 1 WHERE token_type = ? AND token = ?"
	_, err := d.DevicesDB.Exec(ctx, query, tokenType, token)
	if err != nil {
		logx.WithContext(ctx).Errorf("UnregisterDevice error: %v", err)
	}
	return err
}

func JoinInt64s(ids []int64) string {
	if len(ids) == 0 {
		return ""
	}
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(s, ",")
}
