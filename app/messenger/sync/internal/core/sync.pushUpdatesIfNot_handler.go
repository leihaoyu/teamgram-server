/*
 * Created from 'scheme.tl' by 'mtprotoc'
 *
 * Copyright (c) 2021-present,  Teamgram Studio (https://teamgram.io).
 *  All rights reserved.
 *
 * Author: teamgramio (teamgram.io@gmail.com)
 */

package core

import (
	"github.com/teamgram/proto/mtproto"
	chatpb "github.com/teamgram/teamgram-server/app/service/biz/chat/chat"
	userpb "github.com/teamgram/teamgram-server/app/service/biz/user/user"
	"github.com/teamgram/teamgram-server/app/messenger/sync/internal/dao"
	"github.com/teamgram/teamgram-server/app/messenger/sync/sync"
)

// SyncPushUpdatesIfNot
// sync.pushUpdatesIfNot user_id:long excludes:Vector<int64> updates:Updates = Void;
func (c *SyncCore) SyncPushUpdatesIfNot(in *sync.TLSyncPushUpdatesIfNot) (*mtproto.Void, error) {
	userId := in.GetUserId()
	excludes := in.GetExcludes()

	c.Logger.Infof("sync.pushUpdatesIfNot - userId: %d, excludes: %v", userId, excludes)

	if c.svcCtx.Dao.APNsClient == nil {
		c.Logger.Infof("sync.pushUpdatesIfNot - APNs client not configured, skip")
		return mtproto.EmptyVoid, nil
	}

	// 1. Get all APNs devices for this user
	devices, err := c.svcCtx.Dao.GetUserAPNsDevices(c.ctx, userId)
	if err != nil {
		c.Logger.Errorf("sync.pushUpdatesIfNot - GetUserAPNsDevices(%d) error: %v", userId, err)
		return mtproto.EmptyVoid, nil
	}
	if len(devices) == 0 {
		c.Logger.Infof("sync.pushUpdatesIfNot - no APNs devices for user %d", userId)
		return mtproto.EmptyVoid, nil
	}

	// 2. Build exclude map (online sessions that already got the update)
	excludeMap := make(map[int64]bool, len(excludes))
	for _, id := range excludes {
		excludeMap[id] = true
	}

	// 3. Extract push payload from updates
	pushPayload := extractPushPayload(in.GetUpdates())
	if pushPayload == nil {
		c.Logger.Infof("sync.pushUpdatesIfNot - no pushable content in updates")
		return mtproto.EmptyVoid, nil
	}

	// 4. Resolve sender name from user service if not in updates
	if pushPayload.SenderName == "New Message" && pushPayload.FromUserId > 0 && c.svcCtx.Dao.UserClient != nil {
		userData, err := c.svcCtx.Dao.UserClient.UserGetUserDataById(c.ctx, &userpb.TLUserGetUserDataById{
			UserId: pushPayload.FromUserId,
		})
		if err == nil && userData != nil {
			name := userData.GetFirstName()
			if ln := userData.GetLastName(); ln != "" {
				name += " " + ln
			}
			if name != "" {
				pushPayload.SenderName = name
			}
		}
	}

	// 5. Resolve chat title from chat service if not in updates
	if pushPayload.ChatId > 0 && pushPayload.ChatTitle == "" {
		mutableChat, err := c.svcCtx.Dao.ChatClient.ChatGetMutableChat(c.ctx, &chatpb.TLChatGetMutableChat{
			ChatId: pushPayload.ChatId,
		})
		if err == nil && mutableChat != nil {
			pushPayload.ChatTitle = mutableChat.Title()
		}
	}

	// 6. Send push to each offline APNs device, deduplicate by token
	pushedTokens := make(map[string]bool)
	for _, dev := range devices {
		if excludeMap[dev.AuthKeyId] {
			c.Logger.Infof("sync.pushUpdatesIfNot - skip online device authKeyId: %d", dev.AuthKeyId)
			continue
		}
		if pushedTokens[dev.Token] {
			continue
		}
		pushedTokens[dev.Token] = true

		c.Logger.Infof("sync.pushUpdatesIfNot - sending APNs push to device token: %s...", dev.Token[:min(8, len(dev.Token))])
		if err := c.svcCtx.Dao.SendAPNsPush(c.ctx, dev.Token, pushPayload); err != nil {
			c.Logger.Errorf("sync.pushUpdatesIfNot - SendAPNsPush error: %v", err)
		}
	}

	return mtproto.EmptyVoid, nil
}

func extractPushPayload(updates *mtproto.Updates) *dao.PushPayload {
	if updates == nil {
		return nil
	}

	var result *dao.PushPayload

	// Handle different update container types
	switch updates.PredicateName {
	case mtproto.Predicate_updateShortMessage:
		// Direct user message
		result = &dao.PushPayload{
			FromUserId: updates.UserId,
			Message:    updates.Message,
			MsgId:      updates.Id,
			PeerType:   "user",
			PeerId:     updates.UserId,
		}
	case mtproto.Predicate_updateShortChatMessage:
		// Chat message
		result = &dao.PushPayload{
			FromUserId: updates.FromId,
			Message:    updates.Message,
			MsgId:      updates.Id,
			PeerType:   "chat",
			PeerId:     updates.ChatId,
			ChatId:     updates.ChatId,
		}
	case mtproto.Predicate_updates:
		// Full updates container - look for updateNewMessage
		for _, update := range updates.Updates {
			if update == nil {
				continue
			}
			if update.PredicateName == mtproto.Predicate_updateNewMessage {
				msg := update.Message_MESSAGE
				if msg == nil {
					continue
				}
				p := &dao.PushPayload{
					Message: msg.Message,
					MsgId:   msg.Id,
					Silent:  msg.Silent,
				}
				// Extract sender
				if msg.FromId != nil {
					p.FromUserId = msg.FromId.UserId
				}
				// Extract peer info
				if msg.PeerId != nil {
					switch msg.PeerId.PredicateName {
					case mtproto.Predicate_peerUser:
						p.PeerType = "user"
						p.PeerId = msg.PeerId.UserId
					case mtproto.Predicate_peerChat:
						p.PeerType = "chat"
						p.PeerId = msg.PeerId.ChatId
						p.ChatId = msg.PeerId.ChatId
					case mtproto.Predicate_peerChannel:
						p.PeerType = "channel"
						p.PeerId = msg.PeerId.ChannelId
					}
				}
				result = p
				break
			}
		}
	case mtproto.Predicate_updateShort:
		// Single update - check if it's updateNewMessage
		if updates.Update != nil && updates.Update.PredicateName == mtproto.Predicate_updateNewMessage {
			msg := updates.Update.Message_MESSAGE
			if msg != nil {
				p := &dao.PushPayload{
					Message: msg.Message,
					MsgId:   msg.Id,
					Silent:  msg.Silent,
				}
				if msg.FromId != nil {
					p.FromUserId = msg.FromId.UserId
				}
				if msg.PeerId != nil {
					switch msg.PeerId.PredicateName {
					case mtproto.Predicate_peerUser:
						p.PeerType = "user"
						p.PeerId = msg.PeerId.UserId
					case mtproto.Predicate_peerChat:
						p.PeerType = "chat"
						p.PeerId = msg.PeerId.ChatId
						p.ChatId = msg.PeerId.ChatId
					case mtproto.Predicate_peerChannel:
						p.PeerType = "channel"
						p.PeerId = msg.PeerId.ChannelId
					}
				}
				result = p
			}
		}
	}

	if result == nil {
		return nil
	}

	// Extract sender name from updates.Users
	if result.FromUserId > 0 && len(updates.Users) > 0 {
		for _, u := range updates.Users {
			if u != nil && u.Id == result.FromUserId {
				name := u.GetFirstName().GetValue()
				if ln := u.GetLastName().GetValue(); ln != "" {
					name += " " + ln
				}
				if name != "" {
					result.SenderName = name
				}
				break
			}
		}
	}

	// Extract chat title from updates.Chats
	if result.ChatId > 0 && len(updates.Chats) > 0 {
		for _, chat := range updates.Chats {
			if chat != nil && chat.Id == result.ChatId {
				result.ChatTitle = chat.GetTitle()
				break
			}
		}
	}

	// Default sender name if not found
	if result.SenderName == "" {
		result.SenderName = "New Message"
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
