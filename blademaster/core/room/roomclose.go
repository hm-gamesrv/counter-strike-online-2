package room

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

func OnCloseResultRequest(p *PacketData, client net.Conn) {
	//找到对应用户
	uPtr := GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "try to close result but not in server !")
		return
	}

	//检查房间
	rm := GetRoomFromID(uPtr.GetUserChannelServerID(),
		uPtr.GetUserChannelID(),
		uPtr.GetUserRoomID())
	if rm == nil ||
		rm.Id <= 0 {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "close result window in null room !")
	} else {
		switch rm.Setting.GameModeID { //清除无用房间
		case ModeZd_boss1,
			ModeZd_boss2,
			ModeZd_boss3,
			ModeCampaign1,
			ModeCampaign2,
			ModeCampaign3,
			ModeCampaign4,
			ModeCampaign5,
			ModeCampaign6,
			ModeCampaign7,
			ModeCampaign8,
			ModeCampaign9:
			DelChannelRoom(rm.Id,
				uPtr.GetUserChannelID(),
				uPtr.GetUserChannelServerID())
			uPtr.QuitRoom()
			uPtr.QuitChannel()
		default:
		}
	}
	//发送数据
	rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeHost), BuildCloseResultWindow())
	SendPacket(rst, uPtr.CurrentConnection)
	DebugInfo(2, "User", uPtr.UserName, "closed game result window from room id", uPtr.CurrentRoomId)

	//在线时间奖励或进度等
	//...
}

func BuildCloseResultWindow() []byte {
	buf := make([]byte, 1)
	buf[0] = LeaveResultWindow
	return buf
}
