package disassemble

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const (
	disassembleItem = 2
)

func OnDisassembleItem(p *PacketData, client net.Conn) {
	var pkt InDisassembleItemPacket
	if p.PraseDisassembleItemPacket(&pkt) {
		switch pkt.SubType {
		case disassembleItem:
			OnDisassembleWeapon(p, client)
		default:
			DebugInfo(2, "Unknown disassemble item packet", pkt.SubType, "from", client.RemoteAddr().String())
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal disassemble item packet from", client.RemoteAddr().String())
	}
}
