package useitem

import (
	"math/rand/v2"
	"net"

	. "cso2-server/blademaster/core/inventory"
	. "cso2-server/blademaster/core/message"
	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

const (
	lotto_base       = 1 //银币
	lotto_max        = 6000
	lotto_event_base = 1 //铜币
	lotto_event_max  = 8900
	lotto_gold_base  = 1 //金币
	lotto_gold_max   = 12000
)

func OnItemUse(p *PacketData, client net.Conn) {
	//检索数据包
	var pkt InItemUsePacket
	if !p.PraseItemUsePacket(&pkt) {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent a error pointlottouse packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "try to request use pointlotto but not in server !")
		return
	}
	//发送数据
	itemID := uPtr.GetItemIDBySeq(pkt.ItemSeq)
	switch itemID {
	case 2001: //改名卡
		if NicknameExists(string(pkt.String)) {
			OnSendMessage(uPtr.CurrentSequence, uPtr.CurrentConnection, MessageDialogBox, CSO2_POPUP_NICKNAME_ALREADY_EXIST)
			DebugInfo(2, "User", uPtr.UserName, "try change nickname to", string(pkt.String), "but this name already exists")
			return
		}

		if err := DeleteNicknameIndex(uPtr.IngameName); err != nil {
			OnSendMessage(uPtr.CurrentSequence, uPtr.CurrentConnection, MessageDialogBox, CSO2_UI_RAMDOMBOX_ALERT_000)
			DebugInfo(2, "User", uPtr.UserName, "try change nickname to", string(pkt.String), "failed", err)
			return
		}

		idx, ok := uPtr.DecreaseItem(itemID)
		if !ok {
			OnSendMessage(uPtr.CurrentSequence, uPtr.CurrentConnection, MessageDialogBox, CSO2_UI_RAMDOMBOX_ALERT_000)
			DebugInfo(2, "User", uPtr.UserName, "use item", itemID, "failed")
			return
		}

		rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeInventory_Create),
			BuildInventoryInfoSingle(uPtr, 0, idx))
		SendPacket(rst, uPtr.CurrentConnection)

		uPtr.SetUserName(uPtr.UserName, string(pkt.String))

		rst = BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeUseItem),
			buildNickNameChange())
		SendPacket(rst, uPtr.CurrentConnection)

		DebugInfo(2, "User", uPtr.UserName, "changed nickname to", string(pkt.String))
	case 2008, 2013, 2014: //银币 id 2008，铜币，金币
		idx, ok := uPtr.DecreaseItem(itemID)
		if !ok {
			OnSendMessage(uPtr.CurrentSequence, uPtr.CurrentConnection, MessageDialogBox, CSO2_UI_RAMDOMBOX_ALERT_000)
			DebugInfo(2, "User", uPtr.UserName, "use item", itemID, "failed")
			return
		}

		rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeInventory_Create),
			BuildInventoryInfoSingle(uPtr, 0, idx))
		SendPacket(rst, uPtr.CurrentConnection)

		point := UsePointLotto(itemID)
		uPtr.GetPoints(point)

		rst = BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeUseItem),
			buildUsePoint(uint32(point)))
		SendPacket(rst, uPtr.CurrentConnection)

		DebugInfo(2, "User", uPtr.UserName, "got point", point, "by using pointlotto", itemID)
	default:
		DebugInfo(2, "User", uPtr.UserName, "try using item but itemid is", itemID)
		return
	}

	//UserInfo部分
	rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeUserInfo), BuildUserInfo(0xFFFFFFFF, NewUserInfo(uPtr), uPtr.Userid, true))
	SendPacket(rst, uPtr.CurrentConnection)
}

func buildUsePoint(point uint32) []byte {
	buf := make([]byte, 25)
	offset := 0
	WriteUint8(&buf, useitem, &offset)
	WriteUint8(&buf, 5, &offset)
	WriteUint8(&buf, 1, &offset)      //unk00
	WriteUint32(&buf, 0, &offset)     //unk01
	WriteUint32(&buf, point, &offset) //mpoint
	return buf[:offset]
}

func buildNickNameChange() []byte {
	buf := make([]byte, 25)
	offset := 0
	WriteUint8(&buf, useitem, &offset)
	WriteUint8(&buf, 0, &offset)
	WriteUint8(&buf, 1, &offset)
	return buf[:offset]
}

func UsePointLotto(itemid uint32) uint64 {
	switch itemid {
	case 2008: //银币 id 2008
		return uint64(lotto_base + rand.IntN(lotto_max))
	case 2013: //铜币
		return uint64(lotto_event_base + rand.IntN(lotto_event_max))
	case 2014: //金币
		return uint64(lotto_gold_base + rand.IntN(lotto_gold_max))
	default:
		return 0
	}
}
