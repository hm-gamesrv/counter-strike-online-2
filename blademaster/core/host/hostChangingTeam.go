package host

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

func OnChangingTeam(p *PacketData, client net.Conn) {
	//检索数据包
	var pkt InHostTeamChangingPacket
	if !p.PraseInTeamChangingPacket(&pkt) {
		DebugInfo(2, "Error : User from", client.RemoteAddr().String(), "sent a error TeamChanging packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		DebugInfo(2, "Error : User from", client.RemoteAddr().String(), "sent TeamChanging but not in server !")
		return
	}
	destUser := GetUserFromID(uint32(pkt.UserId))
	if destUser == nil ||
		destUser.Userid <= 0 {
		DebugInfo(2, "Error : User from", client.RemoteAddr().String(), "sent TeamChanging but dester is not in server !")
		return
	}
	//找到房间
	rm := GetRoomFromID(uPtr.GetUserChannelServerID(),
		uPtr.GetUserChannelID(),
		uPtr.CurrentRoomId)
	if rm == nil ||
		rm.Id <= 0 {
		DebugInfo(2, "Error : User from", client.RemoteAddr().String(), "sent TeamChanging but is not host !")
		return
	}
	//更新数据
	destUser.SetUserTeam(pkt.NewTeam)
	result := BuildChangingTeam(destUser.Userid, destUser.CurrentTeam)

	for _, v := range rm.Users {
		rst := BytesCombine(BuildHeader(v.CurrentSequence, p.Id), result)
		SendPacket(rst, v.CurrentConnection)
	}
	DebugInfo(2, "Host changed User", destUser.UserName, "to team", pkt.NewTeam)
}
func BuildChangingTeam(id uint32, team uint8) []byte {
	buf := make([]byte, 7)
	offset := 0
	WriteUint8(&buf, OUTsetUserTeam, &offset)
	WriteUint8(&buf, 1, &offset) //numberOfUser
	WriteUint32(&buf, id, &offset)
	WriteUint8(&buf, team, &offset)
	return buf[:offset]
}
