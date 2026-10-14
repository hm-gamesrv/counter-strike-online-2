package option

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const (
	SaveBuyMenu = 1
)

func OnOption(p *PacketData, client net.Conn) {
	var pkt InOptionPacket
	if p.PraseOptionPacket(&pkt) {
		switch pkt.OptionPacketType {
		case SaveBuyMenu:
			OnSaveBuyMenu(p, client)
		default:
			DebugInfo(2, "Unknown option packet", pkt.OptionPacketType, "from", client.RemoteAddr().String())
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal option packet from", client.RemoteAddr().String())
	}
}
