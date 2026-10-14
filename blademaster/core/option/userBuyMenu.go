package option

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

func OnSaveBuyMenu(p *PacketData, client net.Conn) {
	var pkt InOptionBuyMenu
	if !p.PraseSaveBuyMenu(&pkt) {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent a illegal save buymenu packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "try to save buymenu but not in server !")
		return
	}
	//修改数据
	uPtr.SetBuyMenu(pkt.Buymenu)
	DebugInfo(1, "User", uPtr.UserName, "updated buymenu ...")
}
