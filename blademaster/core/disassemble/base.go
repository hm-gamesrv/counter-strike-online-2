package disassemble

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const (
	disassemble = 1
)

func OnDisassemble(p *PacketData, client net.Conn) {
	var pkt InDisassemblePacket
	if p.PraseDisassemblePacket(&pkt) {
		switch pkt.Type {
		case disassemble:
			OnDisassembleItem(p, client)
		default:
			DebugInfo(2, "Unknown disassemble packet", pkt.Type, "from", client.RemoteAddr().String())
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal disassemble packet from", client.RemoteAddr().String())
	}
}
