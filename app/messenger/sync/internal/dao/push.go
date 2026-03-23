package dao

import (
	"context"
	"fmt"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeviceInfo struct {
	AuthKeyId  int64  `db:"auth_key_id"`
	Token      string `db:"token"`
	AppSandbox bool   `db:"app_sandbox"`
	NoMuted    bool   `db:"no_muted"`
}

func (d *Dao) GetUserAPNsDevices(ctx context.Context, userId int64) ([]DeviceInfo, error) {
	if d.DevicesDB == nil {
		return nil, nil
	}

	var devices []DeviceInfo
	query := "SELECT auth_key_id, token, app_sandbox, no_muted FROM devices WHERE user_id = ? AND token_type = 1 AND state = 0"
	err := d.DevicesDB.QueryRowsPartial(ctx, &devices, query, userId)
	if err != nil {
		logx.WithContext(ctx).Errorf("GetUserAPNsDevices(%d) error: %v", userId, err)
		return nil, err
	}

	return devices, nil
}

type PushPayload struct {
	SenderName string
	Message    string
	FromUserId int64
	MsgId      int32
	PeerType   string // "user", "chat", "channel"
	PeerId     int64
	ChatId     int64
	ChatTitle  string // 群聊名称
	Silent     bool
}

func (d *Dao) SendAPNsPush(ctx context.Context, deviceToken string, p *PushPayload) error {
	if d.APNsClient == nil {
		logx.WithContext(ctx).Errorf("SendAPNsPush: APNs client not initialized")
		return fmt.Errorf("APNs client not initialized")
	}

	pl := payload.NewPayload()

	if p.Silent {
		pl.ContentAvailable()
	} else {
		const maxBodyLen = 120

		switch p.PeerType {
		case "chat", "channel":
			// 群聊: title=群名, body="发送者: 消息内容"
			title := p.ChatTitle
			if title == "" {
				title = "Group"
			}
			pl.AlertTitle(title)

			body := p.SenderName + ": " + p.Message
			if len([]rune(body)) > maxBodyLen {
				body = string([]rune(body)[:maxBodyLen]) + "..."
			}
			pl.AlertBody(body)
		default:
			// 私聊: title=发送者昵称, body=消息内容
			pl.AlertTitle(p.SenderName)

			body := p.Message
			if len([]rune(body)) > maxBodyLen {
				body = string([]rune(body)[:maxBodyLen]) + "..."
			}
			pl.AlertBody(body)
		}

		pl.Sound("default")
		pl.Badge(1)
	}
	pl.MutableContent()

	// Custom data for the client
	customData := map[string]interface{}{
		"from_id":   p.FromUserId,
		"msg_id":    p.MsgId,
		"peer_type": p.PeerType,
		"peer_id":   p.PeerId,
	}
	if p.ChatId > 0 {
		customData["chat_id"] = p.ChatId
	}

	pl.Custom("custom", customData)

	notification := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       d.APNsBundleID,
		Payload:     pl,
	}

	resp, err := d.APNsClient.PushWithContext(ctx, notification)
	if err != nil {
		logx.WithContext(ctx).Errorf("SendAPNsPush error: %v", err)
		return err
	}

	if !resp.Sent() {
		logx.WithContext(ctx).Errorf("SendAPNsPush failed: %d %s %s", resp.StatusCode, resp.ApnsID, resp.Reason)
		// If token is invalid, mark it as unregistered
		if resp.Reason == apns2.ReasonBadDeviceToken ||
			resp.Reason == apns2.ReasonUnregistered ||
			resp.Reason == apns2.ReasonExpiredToken {
			logx.WithContext(ctx).Infof("SendAPNsPush: marking invalid token as unregistered: %s", deviceToken)
			if d.DevicesDB != nil {
				d.DevicesDB.Exec(ctx, "UPDATE devices SET state = 1 WHERE token_type = 1 AND token = ?", deviceToken)
			}
		}
		return fmt.Errorf("APNs push failed: %s", resp.Reason)
	}

	logx.WithContext(ctx).Infof("SendAPNsPush success: %s", resp.ApnsID)
	return nil
}
