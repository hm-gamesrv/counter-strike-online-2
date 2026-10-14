package useitem

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const (
	active     = 0
	tryuseitem = 1
	useitem    = 2
)

func OnUseItem(p *PacketData, client net.Conn) {
	var pkt InPointLottoPacket
	if p.PrasePointLottoPacket(&pkt) {
		switch pkt.Type {
		case active:
		case useitem:
			OnItemUse(p, client)
		case tryuseitem:
			OnTryItemUse(p, client)
		default:
			DebugInfo(2, "Unknown useitem packet", pkt.Type, "from", client.RemoteAddr().String())
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal useitem packet from", client.RemoteAddr().String())
	}
}
