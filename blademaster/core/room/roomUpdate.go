package room

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

// onUpdateRoom 房主更新房间信息
func OnUpdateRoom(p *PacketData, client net.Conn) {
	//检索数据报
	var pkt InUpSettingReq
	if !p.PraseUpdateRoomPacket(&pkt) {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent a illegal packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "try to toggle ready status but not in server !")
		return
	}
	//检查用户是不是房主
	curroom := GetRoomFromID(uPtr.GetUserChannelServerID(),
		uPtr.GetUserChannelID(),
		uPtr.GetUserRoomID())
	if curroom == nil ||
		curroom.Id <= 0 {
		DebugInfo(2, "Error : User:", uPtr.UserName, "try to update a null room in server !")
		return
	}
	if curroom.HostUserID != uPtr.Userid {
		DebugInfo(2, "Error : User:", uPtr.UserName, "try to update a room but isn't host !")
		return
	}
	//检查用户所在房间
	if curroom.Id != uPtr.CurrentRoomId {
		DebugInfo(2, "Error : User:", uPtr.UserName, "try to update a room but not in !")
		return
	}
	//检查当前是不是正在倒计时
	if curroom.IsGlobalCountdownInProgress() {
		DebugInfo(2, "Error : User:", uPtr.UserName, "try to update a room but is counting !")
		return
	}
	//更新房间设置
	curroom.ToUpdateSetting(&pkt)
	//向房间所有玩家发送更新报文
	settingpkt := BuildRoomSetting(curroom, pkt.Flags)
	for _, v := range curroom.Users {
		rst := BytesCombine(BuildHeader(v.CurrentSequence, PacketTypeRoom), settingpkt)
		SendPacket(rst, v.CurrentConnection)
	}
	DebugInfo(2, "Host", uPtr.UserName, "updated room", string(curroom.Setting.RoomName), "id", curroom.Id)
}
