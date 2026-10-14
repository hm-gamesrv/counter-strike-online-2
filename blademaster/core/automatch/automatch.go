package automatch

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

func OnAutoMatch(p *PacketData, client net.Conn) {
	DebugInfo(2, "Unknown AutoMatch packet from", client.RemoteAddr().String())
}
