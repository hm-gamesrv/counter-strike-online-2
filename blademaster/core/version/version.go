package version

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
)

func OnVersionPacket(seq *uint8, client net.Conn) {
	header := BuildHeader(seq, PacketTypeVersion)
	header[1] = 0 //seq to 0
	*seq = 0
	IsBadHash := make([]byte, 1)
	IsBadHash[0] = 0
	hash := []byte("6246015df9a7d1f7311f888e7e861f18")
	rst := BytesCombine(header, IsBadHash, hash)
	SendPacket(rst, client)
	DebugInfo(1, "Sent a version reply to", client.RemoteAddr().String())
}
