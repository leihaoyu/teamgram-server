package core

import (
	"math/rand"

	"github.com/teamgram/proto/mtproto"
	msgpb "github.com/teamgram/teamgram-server/app/messenger/msg/msg/msg"
	dialogpb "github.com/teamgram/teamgram-server/app/service/biz/dialog/dialog"
)

// MessagesSetChatTheme
// messages.setChatTheme#e63be13f peer:InputPeer emoticon:string = Updates;
func (c *ThemesCore) MessagesSetChatTheme(in *mtproto.TLMessagesSetChatTheme) (*mtproto.Updates, error) {
	peer := mtproto.FromInputPeer2(c.MD.UserId, in.GetPeer())

	// 1. Persist theme emoticon in dialog for both sides
	_, err := c.svcCtx.Dao.DialogClient.DialogSetChatTheme(c.ctx, &dialogpb.TLDialogSetChatTheme{
		UserId:        c.MD.UserId,
		PeerType:      peer.PeerType,
		PeerId:        peer.PeerId,
		ThemeEmoticon: in.GetEmoticon(),
	})
	if err != nil {
		c.Logger.Errorf("messages.setChatTheme - DialogSetChatTheme error: %v", err)
		return nil, err
	}

	// 2. Send service message (messageActionSetChatTheme)
	serviceMessage := mtproto.MakeSetChatThemeService(c.MD.UserId, peer, in.GetEmoticon())

	replyUpdates, err := c.svcCtx.Dao.MsgClient.MsgSendMessage(c.ctx, &msgpb.TLMsgSendMessage{
		UserId:    c.MD.UserId,
		AuthKeyId: c.MD.AuthId,
		PeerType:  peer.PeerType,
		PeerId:    peer.PeerId,
		Message: msgpb.MakeTLOutboxMessage(&msgpb.OutboxMessage{
			NoWebpage:    true,
			Background:   false,
			RandomId:     rand.Int63(),
			Message:      serviceMessage,
			ScheduleDate: nil,
		}).To_OutboxMessage(),
	})
	if err != nil {
		c.Logger.Errorf("messages.setChatTheme - MsgSendMessage error: %v", err)
		return nil, err
	}

	return replyUpdates, nil
}
