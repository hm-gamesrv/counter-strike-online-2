package playerinfo

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/kerlong/encode"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

func OnSetSignature(p *PacketData, client net.Conn) {
	var pkt InSetSignaturePacket
	if !p.PraseSetSignaturePacket(&pkt) {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent a illegal SetSignature packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "try to SetSignature but not in server !")
		return
	}
	//修改数据
	uPtr.SetSignature(pkt.Signature)
	//发送数据包
	rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeUserInfo), BuildSetSignaturePacket(uPtr.Userid, pkt.Signature, pkt.Len))
	SendPacket(rst, uPtr.CurrentConnection)
	DebugInfo(1, "User", uPtr.UserName, "Set Signature to", string(pkt.Signature))
	//如果是在房间内
}

func BuildSetSignaturePacket(id uint32, Signature []byte, len uint8) []byte {
	buf := make([]byte, 10+2*len)
	offset := 0
	WriteUint32(&buf, id, &offset)
	WriteUint32(&buf, 0x40000, &offset)
	ansiString, _ := Utf8ToLocal(string(Signature))
	WriteString(&buf, []byte(ansiString), &offset)
	return buf[:offset]
}
