package host

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

func OnHostGameScorePacket(p *PacketData, client net.Conn) {
	//检索数据包
	var pkt InGameScorePacket
	if !p.PraseInGameScorePacket(&pkt) {
		DebugInfo(2, "Error : User from", client.RemoteAddr().String(), "sent a error GameScore packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		DebugInfo(2, "Error : User from", client.RemoteAddr().String(), "sent GameScore but not in server !")
		return
	}
	//找到对应房间
	rm := GetRoomFromID(uPtr.GetUserChannelServerID(),
		uPtr.GetUserChannelID(),
		uPtr.CurrentRoomId)
	if rm == nil ||
		rm.Id <= 0 {
		return
	}
	//修改房间结果
	rm.SetRoomScore(pkt.CtScore, pkt.TrScore)
	rm.SetRoomWinner(pkt.WinnerTeam)
}
