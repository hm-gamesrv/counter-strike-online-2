package report

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const (
	searchUser = 0
	reportMeg  = 1
)

func OnReportUser(p *PacketData, client net.Conn) {
	var pkt InReportPacket
	if p.PraseReportPacket(&pkt) {
		switch pkt.Type {
		case searchUser:
			OnReportSearchUser(p, client)
		case reportMeg:
			OnReportMsg(p, client)
		default:
			DebugInfo(2, "Unknown report user packet", pkt.Type, "from", client.RemoteAddr().String(), p.Data)
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal report user packet from", client.RemoteAddr().String())
	}
}
