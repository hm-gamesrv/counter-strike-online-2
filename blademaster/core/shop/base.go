package shop

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const (
	outshoplist = 0
	buyitem     = 1
	requestList = 3
)

func OnShopRequest(p *PacketData, client net.Conn) {
	var pkt InShopPacket
	if p.PraseShopPacket(&pkt) {
		switch pkt.InShopType {
		case requestList:
			OnShopList(p, client)
		case buyitem:
			OnShopBuyItem(p, client)
		default:
			DebugInfo(2, "Unknown shop packet", pkt.InShopType, "from", client.RemoteAddr().String())
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal shop packet from", client.RemoteAddr().String())
	}
}
