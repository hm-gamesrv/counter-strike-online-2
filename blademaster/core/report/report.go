package report

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const (
	reportUser = 0
)

func OnReportRequest(p *PacketData, client net.Conn) {
	var pkt InReportPacket
	if p.PraseReportPacket(&pkt) {
		switch pkt.Type {
		case reportUser:
			OnReportUser(p, client)
		default:
			DebugInfo(2, "Unknown report packet", pkt.Type, "from", client.RemoteAddr().String(), p.Data)
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal report packet from", client.RemoteAddr().String())
	}
}
