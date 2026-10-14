package host

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

func OnHostDeathPacket(p *PacketData, client net.Conn) {
	//检索数据包
	var pkt InDeathPacket
	if !p.PraseInDeathPacket(&pkt) {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent a error HostDeath packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromID(pkt.DeadID)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		return
	}
	//修改玩家当前数据
	uPtr.CountDeadNum(pkt.DeathNum)

}
