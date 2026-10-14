package host

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

func OnHostKillPacket(p *PacketData, client net.Conn) {
	//检索数据包
	var pkt InKillPacket
	if !p.PraseInKillPacket(&pkt) {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent a error HostKill packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromID(pkt.KillerID)
	if uPtr != nil &&
		uPtr.Userid > 0 {
		//修改玩家当前数据
		uPtr.CountKillNum(pkt.KillNum)
	}
	//修改房间数据
	uPtr = GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		return
	}
	rm := GetRoomFromID(uPtr.GetUserChannelServerID(),
		uPtr.GetUserChannelID(),
		uPtr.GetUserRoomID())
	if rm == nil ||
		rm.Id < 0 {
		return
	}
	if pkt.PlayerTeam == UserForceCounterTerrorist {
		rm.CountRoomCtKill()
	} else {
		rm.CountRoomTrKill()
	}

}
