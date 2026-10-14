package chat

import (
	"net"

	. "cso2-server/blademaster/core/inventory"
	. "cso2-server/blademaster/core/message"
	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

func OnChatChannelMessage(p *InChatPacket, client net.Conn) {
	//找到对应用户
	uPtr := GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent ChannelMessage but not in server !")
		return
	}
	//找到对应频道
	chlsrv := GetChannelServerWithID(uPtr.GetUserChannelServerID())
	if chlsrv == nil || chlsrv.ServerIndex <= 0 {
		DebugInfo(2, "Error : User", string(uPtr.IngameName), "sent ChannelMessage but not in channelserver !")
		OnSendMessage(uPtr.CurrentSequence, uPtr.CurrentConnection, MessageNotice, GAME_CHANNEL_MESSAGE_NOT_IN)
		return
	}
	chl := GetChannelWithID(uPtr.GetUserChannelID(), chlsrv)
	if chl == nil || chl.ChannelID <= 0 {
		DebugInfo(2, "Error : User", string(uPtr.IngameName), "sent ChannelMessage but not in channel !")
		OnSendMessage(uPtr.CurrentSequence, uPtr.CurrentConnection, MessageNotice, GAME_CHANNEL_MESSAGE_NOT_IN)
		return
	}
	//发送数据

	if string(p.Message) == "/addallitems" {
		if !uPtr.IsGM() {
			OnSendMessage(uPtr.CurrentSequence, uPtr.CurrentConnection, MessageNotice, GAME_GM_NO_AUTHORIZE)
			return
		}
		uPtr.SetInventoryItems(&FullInventoryItem)
		rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeInventory_Add),
			BuildInventoryInfo(uPtr))
		SendPacket(rst, uPtr.CurrentConnection)
		OnSendMessage(uPtr.CurrentSequence, uPtr.CurrentConnection, MessageNotice, GAME_GM_ADD_ALLWEAPONS)
		return
	}

	msg := BuildChannelMessage(uPtr, p)
	for _, v := range UsersManager.SnapshotUsers() {
		if !v.CurrentIsIngame && v.GetUserChannelServerID() == chlsrv.ServerIndex && v.GetUserChannelID() == chl.ChannelID {
			OnSendMessage(uPtr.CurrentSequence, uPtr.CurrentConnection, MessageNotice, msg)
		}
	}
	DebugInfo(1, "User", string(uPtr.IngameName), "say <", string(p.Message), "> in channel", chl.ChannelID, "channelserver", chlsrv.ServerIndex)
}

func BuildChannelMessage(u *User, p *InChatPacket) []byte {
	return BytesCombine([]byte("["+GAME_CHANNEL_MESSAGE+"] "+u.UserName+" : "), p.Message)
}
