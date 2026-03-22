package core

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/authorization/internal/dao"
	msg_client "github.com/teamgram/teamgram-server/app/messenger/msg/msg/client"
	msgpb "github.com/teamgram/teamgram-server/app/messenger/msg/msg/msg"
	chatpb "github.com/teamgram/teamgram-server/app/service/biz/chat/chat"

	"github.com/zeromicro/go-zero/core/contextx"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/status"
)

const maxAutoGroupParticipants = 199 // 200 total minus 1 system admin slot

// welcomeTask holds a pending welcome message to be sent after a delay.
type welcomeTask struct {
	systemAdminId int64
	chatId        int64
	userId        int64
	firstName     string
	groupType     int32
	groupKey      string
	locale        string
}

// autoJoinGroups handles adding a newly registered user to the general group and city group.
// Group creation/joining is synchronous; welcome messages are sent after a delay
// to ensure the user's client session is ready to receive them.
func (c *AuthorizationCore) autoJoinGroups(ctx context.Context, userId int64, firstName string, clientAddr string) {
	if c.svcCtx.Dao.AutoGroupDB == nil {
		c.Logger.Infof("autoJoinGroups: AutoGroupDB not configured, skipping")
		return
	}

	systemAdminId := c.svcCtx.Dao.SystemAdminUserId

	c.Logger.Infof("autoJoinGroups: userId=%d, clientAddr=%s", userId, clientAddr)

	var welcomeTasks []welcomeTask

	// 1. Join general group (synchronous)
	if task := c.joinAutoGroup(ctx, userId, firstName, systemAdminId, dao.AutoGroupTypeGeneral, "", "en"); task != nil {
		welcomeTasks = append(welcomeTasks, *task)
	}

	// 避免连续建群触发 FLOOD_WAIT
	time.Sleep(1 * time.Second)

	// 2. Join city group based on IP geolocation (synchronous)
	cityName, locale := c.svcCtx.Dao.GetCityAndLocaleByIp(clientAddr)
	if cityName == "" && c.svcCtx.Dao.TestCityName != "" {
		cityName = c.svcCtx.Dao.TestCityName
		locale = c.svcCtx.Dao.TestCityLocale
		if locale == "" {
			locale = "en"
		}
		c.Logger.Infof("autoJoinGroups: using test city: %s, locale: %s", cityName, locale)
	}
	if cityName != "" {
		if task := c.joinAutoGroup(ctx, userId, firstName, systemAdminId, dao.AutoGroupTypeCity, cityName, locale); task != nil {
			welcomeTasks = append(welcomeTasks, *task)
		}
	}

	// 3. Send welcome messages after a delay in a goroutine,
	//    giving the client time to establish its session and be ready to receive inbox updates.
	if len(welcomeTasks) > 0 {
		asyncCtx := contextx.ValueOnlyFrom(ctx)
		msgClient := c.svcCtx.Dao.MsgClient
		go func() {
			time.Sleep(3 * time.Second)
			for _, task := range welcomeTasks {
				sendWelcomeMessage(asyncCtx, msgClient, task)
			}
		}()
	}
}

// joinAutoGroup handles the logic of finding or creating an auto group and adding the user.
// Returns a welcomeTask if a welcome message should be sent, nil otherwise.
func (c *AuthorizationCore) joinAutoGroup(
	ctx context.Context,
	userId int64,
	firstName string,
	systemAdminId int64,
	groupType int32,
	groupKey string,
	locale string,
) *welcomeTask {
	db := c.svcCtx.Dao.AutoGroupDB
	var task *welcomeTask

	tR := sqlx.TxWrapper(ctx, db, func(tx *sqlx.Tx, result *sqlx.StoreResult) {
		// Get the current active (non-full) auto group with row lock
		currentGroup, err := c.svcCtx.Dao.GetCurrentAutoGroupTx(tx, groupType, groupKey)
		if err != nil {
			c.Logger.Errorf("joinAutoGroup: GetCurrentAutoGroupTx error: %v", err)
			result.Err = err
			return
		}

		if currentGroup == nil {
			// No active group exists — system admin creates a new group, user joins as member
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
				CreatorUserId:    systemAdminId,
				ParticipantCount: 1,
			})
			if err != nil {
				c.Logger.Errorf("joinAutoGroup: CreateAutoGroupTx error: %v", err)
				result.Err = err
				return
			}

			task = &welcomeTask{
				systemAdminId: systemAdminId,
				chatId:        chatId,
				userId:        userId,
				firstName:     firstName,
				groupType:     groupType,
				groupKey:      groupKey,
				locale:        locale,
			}
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
					task = c.handleGroupFull(ctx, tx, userId, firstName, systemAdminId, groupType, groupKey, locale, currentGroup)
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

			task = &welcomeTask{
				systemAdminId: systemAdminId,
				chatId:        currentGroup.ChatId,
				userId:        userId,
				firstName:     firstName,
				groupType:     groupType,
				groupKey:      groupKey,
				locale:        locale,
			}
		}
	})

	if tR.Err != nil {
		c.Logger.Errorf("joinAutoGroup: transaction error: %v", tR.Err)
		return nil
	}

	return task
}

// handleGroupFull handles the case where ChatAddChatUser fails because the group is full.
// Creates a new group with incremented sequence number for the user.
func (c *AuthorizationCore) handleGroupFull(
	ctx context.Context,
	tx *sqlx.Tx,
	userId int64,
	firstName string,
	systemAdminId int64,
	groupType int32,
	groupKey string,
	locale string,
	currentGroup *dao.AutoGroupDO,
) *welcomeTask {
	// Mark current group as full
	if err := c.svcCtx.Dao.MarkAutoGroupFullTx(tx, currentGroup.ChatId); err != nil {
		c.Logger.Errorf("handleGroupFull: MarkAutoGroupFullTx error: %v", err)
		return nil
	}

	// Find max sequence number
	maxSeq, err := c.svcCtx.Dao.GetMaxSequenceNumTx(tx, groupType, groupKey)
	if err != nil {
		c.Logger.Errorf("handleGroupFull: GetMaxSequenceNumTx error: %v", err)
		return nil
	}

	newSeq := maxSeq + 1
	chatId, err := c.createAutoGroupChat(ctx, userId, systemAdminId, groupType, groupKey, locale, newSeq)
	if err != nil {
		c.Logger.Errorf("handleGroupFull: createAutoGroupChat error: %v", err)
		return nil
	}

	err = c.svcCtx.Dao.CreateAutoGroupTx(tx, &dao.AutoGroupDO{
		GroupType:        groupType,
		GroupKey:         groupKey,
		SequenceNum:      newSeq,
		ChatId:           chatId,
		CreatorUserId:    systemAdminId,
		ParticipantCount: 1,
	})
	if err != nil {
		c.Logger.Errorf("handleGroupFull: CreateAutoGroupTx error: %v", err)
		return nil
	}

	return &welcomeTask{
		systemAdminId: systemAdminId,
		chatId:        chatId,
		userId:        userId,
		firstName:     firstName,
		groupType:     groupType,
		groupKey:      groupKey,
		locale:        locale,
	}
}

// createAutoGroupChat creates a new chat group with system admin as owner and the user as a member.
func (c *AuthorizationCore) createAutoGroupChat(
	ctx context.Context,
	userId int64,
	systemAdminId int64,
	groupType int32,
	groupKey string,
	locale string,
	sequenceNum int32,
) (int64, error) {
	title := makeGroupTitle(groupType, groupKey, locale, sequenceNum)

	chat, err := c.svcCtx.Dao.ChatClient.ChatCreateChat2(ctx, &chatpb.TLChatCreateChat2{
		CreatorId:  systemAdminId,
		UserIdList: []int64{userId},
		Title:      title,
	})
	if err != nil {
		return 0, err
	}

	return chat.Chat.Id, nil
}

// sendWelcomeMessage sends a welcome message from the system admin to the chat.
// This is a standalone function (not a method) so it can be called from a goroutine safely.
func sendWelcomeMessage(ctx context.Context, msgClient msg_client.MsgClient, task welcomeTask) {
	welcomeText := makeWelcomeMessage(task.firstName, task.groupType, task.groupKey, task.locale)

	message := mtproto.MakeTLMessage(&mtproto.Message{
		Out:    true,
		Date:   int32(time.Now().Unix()),
		FromId: mtproto.MakePeerUser(task.systemAdminId),
		PeerId: mtproto.MakePeerChat(task.chatId),
		Message: welcomeText,
	}).To_Message()

	_, err := msgClient.MsgSendMessage(ctx, &msgpb.TLMsgSendMessage{
		UserId:    task.systemAdminId,
		AuthKeyId: 0,
		PeerType:  mtproto.PEER_CHAT,
		PeerId:    task.chatId,
		Message: msgpb.MakeTLOutboxMessage(&msgpb.OutboxMessage{
			NoWebpage:  true,
			Background: false,
			RandomId:   rand.Int63(),
			Message:    message,
		}).To_OutboxMessage(),
	})
	if err != nil {
		logx.WithContext(ctx).Errorf("sendWelcomeMessage: MsgSendMessage error: %v", err)
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
