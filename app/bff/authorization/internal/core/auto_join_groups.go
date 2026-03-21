package core

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/authorization/internal/dao"
	msgpb "github.com/teamgram/teamgram-server/app/messenger/msg/msg/msg"
	chatpb "github.com/teamgram/teamgram-server/app/service/biz/chat/chat"

	"google.golang.org/grpc/status"
)

const maxAutoGroupParticipants = 199 // 200 total minus 1 system admin slot

// autoJoinGroups handles adding a newly registered user to the general group and city group.
// This should be called asynchronously after registration completes.
func (c *AuthorizationCore) autoJoinGroups(ctx context.Context, userId int64, firstName string, clientAddr string) {
	if c.svcCtx.Dao.AutoGroupDB == nil {
		c.Logger.Infof("autoJoinGroups: AutoGroupDB not configured, skipping")
		return
	}

	systemAdminId := c.svcCtx.Dao.SystemAdminUserId

	// 1. Join general group
	c.joinAutoGroup(ctx, userId, firstName, systemAdminId, dao.AutoGroupTypeGeneral, "", "en")

	// 2. Join city group based on IP geolocation
	cityName, locale := c.svcCtx.Dao.GetCityAndLocaleByIp(clientAddr)
	if cityName != "" {
		c.joinAutoGroup(ctx, userId, firstName, systemAdminId, dao.AutoGroupTypeCity, cityName, locale)
	}
}

// joinAutoGroup handles the logic of finding or creating an auto group and adding the user.
func (c *AuthorizationCore) joinAutoGroup(
	ctx context.Context,
	userId int64,
	firstName string,
	systemAdminId int64,
	groupType int32,
	groupKey string,
	locale string,
) {
	db := c.svcCtx.Dao.AutoGroupDB

	tR := sqlx.TxWrapper(ctx, db, func(tx *sqlx.Tx, result *sqlx.StoreResult) {
		// Get the current active (non-full) auto group with row lock
		currentGroup, err := c.svcCtx.Dao.GetCurrentAutoGroupTx(tx, groupType, groupKey)
		if err != nil {
			c.Logger.Errorf("joinAutoGroup: GetCurrentAutoGroupTx error: %v", err)
			result.Err = err
			return
		}

		if currentGroup == nil {
			// No active group exists — this user becomes the creator of a new group
			chatId, err := c.createAutoGroupChat(ctx, userId, systemAdminId, groupType, groupKey, locale, 1)
			if err != nil {
				c.Logger.Errorf("joinAutoGroup: createAutoGroupChat error: %v", err)
				result.Err = err
				return
			}

			// Record the new auto group
			err = c.svcCtx.Dao.CreateAutoGroupTx(tx, &dao.AutoGroupDO{
				GroupType:        groupType,
				GroupKey:         groupKey,
				SequenceNum:      1,
				ChatId:           chatId,
				CreatorUserId:    userId,
				ParticipantCount: 1,
			})
			if err != nil {
				c.Logger.Errorf("joinAutoGroup: CreateAutoGroupTx error: %v", err)
				result.Err = err
				return
			}
			// No welcome message for the group creator
		} else {
			// Active group exists — add the user to it
			_, err := c.svcCtx.Dao.ChatClient.ChatAddChatUser(ctx, &chatpb.TLChatAddChatUser{
				ChatId:    currentGroup.ChatId,
				InviterId: 0, // admin-level add, bypasses privacy checks
				UserId:    userId,
				IsBot:     false,
			})
			if err != nil {
				c.Logger.Errorf("joinAutoGroup: ChatAddChatUser error: %v", err)
				// If the chat is full, create a new group for this user
				if isGroupFullError(err) {
					c.handleGroupFull(ctx, tx, userId, systemAdminId, groupType, groupKey, locale, currentGroup)
					return
				}
				result.Err = err
				return
			}

			// Increment count and check if full
			newCount, err := c.svcCtx.Dao.IncrParticipantCountTx(tx, currentGroup.ChatId)
			if err != nil {
				c.Logger.Errorf("joinAutoGroup: IncrParticipantCountTx error: %v", err)
				result.Err = err
				return
			}

			if newCount >= maxAutoGroupParticipants {
				err = c.svcCtx.Dao.MarkAutoGroupFullTx(tx, currentGroup.ChatId)
				if err != nil {
					c.Logger.Errorf("joinAutoGroup: MarkAutoGroupFullTx error: %v", err)
					result.Err = err
					return
				}
			}

			// Send welcome message from system admin
			c.sendAutoGroupWelcome(ctx, systemAdminId, currentGroup.ChatId, userId, firstName, groupType, groupKey, locale)
		}
	})

	if tR.Err != nil {
		c.Logger.Errorf("joinAutoGroup: transaction error: %v", tR.Err)
	}
}

// handleGroupFull handles the case where ChatAddChatUser fails because the group is full.
// Creates a new group with incremented sequence number for the user.
func (c *AuthorizationCore) handleGroupFull(
	ctx context.Context,
	tx *sqlx.Tx,
	userId int64,
	systemAdminId int64,
	groupType int32,
	groupKey string,
	locale string,
	currentGroup *dao.AutoGroupDO,
) {
	// Mark current group as full
	if err := c.svcCtx.Dao.MarkAutoGroupFullTx(tx, currentGroup.ChatId); err != nil {
		c.Logger.Errorf("handleGroupFull: MarkAutoGroupFullTx error: %v", err)
		return
	}

	// Find max sequence number
	maxSeq, err := c.svcCtx.Dao.GetMaxSequenceNumTx(tx, groupType, groupKey)
	if err != nil {
		c.Logger.Errorf("handleGroupFull: GetMaxSequenceNumTx error: %v", err)
		return
	}

	newSeq := maxSeq + 1
	chatId, err := c.createAutoGroupChat(ctx, userId, systemAdminId, groupType, groupKey, locale, newSeq)
	if err != nil {
		c.Logger.Errorf("handleGroupFull: createAutoGroupChat error: %v", err)
		return
	}

	err = c.svcCtx.Dao.CreateAutoGroupTx(tx, &dao.AutoGroupDO{
		GroupType:        groupType,
		GroupKey:         groupKey,
		SequenceNum:      newSeq,
		ChatId:           chatId,
		CreatorUserId:    userId,
		ParticipantCount: 1,
	})
	if err != nil {
		c.Logger.Errorf("handleGroupFull: CreateAutoGroupTx error: %v", err)
	}
}

// createAutoGroupChat creates a new chat group with the user as creator and system admin as a member.
func (c *AuthorizationCore) createAutoGroupChat(
	ctx context.Context,
	creatorUserId int64,
	systemAdminId int64,
	groupType int32,
	groupKey string,
	locale string,
	sequenceNum int32,
) (int64, error) {
	title := makeGroupTitle(groupType, groupKey, locale, sequenceNum)

	chat, err := c.svcCtx.Dao.ChatClient.ChatCreateChat2(ctx, &chatpb.TLChatCreateChat2{
		CreatorId:  creatorUserId,
		UserIdList: []int64{systemAdminId},
		Title:      title,
	})
	if err != nil {
		return 0, err
	}

	return chat.Chat.Id, nil
}

// sendAutoGroupWelcome sends a welcome message from the system admin to the chat.
func (c *AuthorizationCore) sendAutoGroupWelcome(
	ctx context.Context,
	systemAdminId int64,
	chatId int64,
	newUserId int64,
	firstName string,
	groupType int32,
	groupKey string,
	locale string,
) {
	welcomeText := makeWelcomeMessage(firstName, groupType, groupKey, locale)

	message := mtproto.MakeTLMessage(&mtproto.Message{
		Out:     true,
		Date:    int32(time.Now().Unix()),
		FromId:  mtproto.MakePeerUser(systemAdminId),
		PeerId:  mtproto.MakePeerChat(chatId),
		Message: welcomeText,
	}).To_Message()

	_, err := c.svcCtx.Dao.MsgClient.MsgSendMessage(ctx, &msgpb.TLMsgSendMessage{
		UserId:    systemAdminId,
		AuthKeyId: 0,
		PeerType:  mtproto.PEER_CHAT,
		PeerId:    chatId,
		Message: msgpb.MakeTLOutboxMessage(&msgpb.OutboxMessage{
			NoWebpage:  true,
			Background: false,
			RandomId:   rand.Int63(),
			Message:    message,
		}).To_OutboxMessage(),
	})
	if err != nil {
		c.Logger.Errorf("sendAutoGroupWelcome: MsgSendMessage error: %v", err)
	}
}

// makeGroupTitle generates the group title based on type, key, and sequence number.
func makeGroupTitle(groupType int32, groupKey string, locale string, sequenceNum int32) string {
	if groupType == dao.AutoGroupTypeGeneral {
		if sequenceNum <= 1 {
			return "总群"
		}
		return fmt.Sprintf("总群 %d", sequenceNum)
	}

	// City group
	suffix := ""
	if sequenceNum > 1 {
		suffix = fmt.Sprintf(" %d", sequenceNum)
	}

	// Use locale-appropriate group suffix
	switch locale {
	case "ja":
		return fmt.Sprintf("%sグループ%s", groupKey, suffix)
	case "de":
		return fmt.Sprintf("%s-Gruppe%s", groupKey, suffix)
	case "es":
		return fmt.Sprintf("Grupo %s%s", groupKey, suffix)
	case "fr":
		return fmt.Sprintf("Groupe %s%s", groupKey, suffix)
	case "pt-BR":
		return fmt.Sprintf("Grupo %s%s", groupKey, suffix)
	case "ru":
		return fmt.Sprintf("Группа %s%s", groupKey, suffix)
	default:
		// zh-CN, en, and others
		return fmt.Sprintf("%s群%s", groupKey, suffix)
	}
}

// makeWelcomeMessage generates a locale-appropriate welcome message.
func makeWelcomeMessage(firstName string, groupType int32, groupKey string, locale string) string {
	if groupType == dao.AutoGroupTypeGeneral {
		return fmt.Sprintf("欢迎新小伙伴 %s 加入大家庭！有什么问题随时在群里聊~", firstName)
	}

	// City group: locale-specific welcome messages with local flavor
	switch locale {
	case "zh-CN":
		return fmt.Sprintf("欢迎 %s 来到%s群！在这里认识更多本地的小伙伴吧~", firstName, groupKey)
	case "ja":
		return fmt.Sprintf("%sさん、%sグループへようこそ！地元の仲間と繋がりましょう～", firstName, groupKey)
	case "de":
		return fmt.Sprintf("Willkommen %s in der %s-Gruppe! Lerne Leute aus der Gegend kennen~", firstName, groupKey)
	case "es":
		return fmt.Sprintf("¡Bienvenido/a %s al grupo de %s! Conecta con gente de aquí~", firstName, groupKey)
	case "fr":
		return fmt.Sprintf("Bienvenue %s dans le groupe de %s ! Faites connaissance ici~", firstName, groupKey)
	case "pt-BR":
		return fmt.Sprintf("Bem-vindo/a %s ao grupo de %s! Conheça pessoal da região~", firstName, groupKey)
	case "ru":
		return fmt.Sprintf("Добро пожаловать, %s, в группу %s! Общайтесь с местными~", firstName, groupKey)
	default:
		return fmt.Sprintf("Welcome %s to the %s group! Connect with locals here~", firstName, groupKey)
	}
}

// isGroupFullError checks if an error indicates the group is full.
func isGroupFullError(err error) bool {
	// The chat service returns ErrUsersTooFew (reused error code) when chat has >= 200 participants
	if s, ok := status.FromError(err); ok {
		return s.Message() == "USERS_TOO_FEW"
	}
	return false
}
