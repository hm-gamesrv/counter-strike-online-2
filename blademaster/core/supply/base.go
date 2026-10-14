package supply

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const (
	requestList = 0
	openbox     = 1
)

func OnSupplyRequest(p *PacketData, client net.Conn) {
	var pkt InSupplyPacket
	if p.PraseSupplyPacket(&pkt) {
		switch pkt.Type {
		case requestList:
			OnSupplyList(p, client)
		case openbox:
			OnSupplyOpenBox(p, client)
		default:
			DebugInfo(2, "Unknown supply packet", pkt.Type, "from", client.RemoteAddr().String())
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal shop packet from", client.RemoteAddr().String())
	}
}
