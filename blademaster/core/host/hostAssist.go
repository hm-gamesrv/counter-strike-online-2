package host

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

func OnHostAssistPacket(p *PacketData, client net.Conn) {
	//检索数据包
	var pkt InAssistPacket
	if !p.PraseInAssistPacket(&pkt) {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent a error HostKill packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromID(pkt.AssisterID)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		return
	}
	//修改玩家当前数据
	uPtr.CountAssistNum()
}
