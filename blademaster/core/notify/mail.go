package notify

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const (
	list = 0
)

func OnNotify(p *PacketData, client net.Conn) {
	var pkt InNotifyPacket
	if p.PraseNotifyPacket(&pkt) {
		switch pkt.InNotifyType {
		case list:
			OnNotifyList(p, client)
		default:
			DebugInfo(2, "Unknown notify packet", pkt.InNotifyType, "from", client.RemoteAddr().String())
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal notify packet from", client.RemoteAddr().String(), p.Data)
	}
}
