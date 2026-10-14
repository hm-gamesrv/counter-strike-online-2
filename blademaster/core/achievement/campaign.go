package achievement

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

var (
	RewardCapmgaign1 = []OutAchievementCampaignItems{}
	RewardCapmgaign2 = []OutAchievementCampaignItems{}
	RewardCapmgaign3 = []OutAchievementCampaignItems{}
	RewardCapmgaign4 = []OutAchievementCampaignItems{}
	RewardCapmgaign5 = []OutAchievementCampaignItems{}
)

func OnAchievementCampaign(p *PacketData, client net.Conn) {
	//检索数据包
	var pkt InAchievementCampaignPacket
	if !p.PraseInAchievementCampaignPacket(&pkt) {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent a error AchievementCampaign packet !")
		return
	}
	//找到对应用户
	uPtr := GetUserFromConnection(client)
	if uPtr == nil ||
		uPtr.Userid <= 0 {
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent AchievementCampaign packet but not in server !")
		return
	}
	//处理关卡
	switch pkt.CampaignId {
	case Campaign_0:
		reward := OutAchievementCampaign{}
		rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeAchievement), BuildAchievementCampaign(0, &reward, pkt.CampaignId))
		SendPacket(rst, uPtr.CurrentConnection)
		DebugInfo(1, "User", uPtr.UserName, "get achievement campaign id", pkt.CampaignId)
	case Campaign_1:
		reward := OutAchievementCampaign{
			Unk00:        0,
			Unk01:        0,
			RewardTitle:  0,
			RewardIcon:   0,
			RewardPoints: 0,
			RewardXp:     3000,
			NumOfItems:   uint8(len(RewardCapmgaign1)),
			Items:        RewardCapmgaign1,
			Unk02:        0,
		}
		rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeAchievement), BuildAchievementCampaign(0x60, &reward, pkt.CampaignId))
		SendPacket(rst, uPtr.CurrentConnection)
		DebugInfo(1, "User", uPtr.UserName, "get achievement campaign id", pkt.CampaignId)
	case Campaign_2:
		reward := OutAchievementCampaign{
			Unk00:        0,
			Unk01:        0,
			RewardTitle:  0,
			RewardIcon:   0,
			RewardPoints: 5000,
			RewardXp:     0,
			NumOfItems:   uint8(len(RewardCapmgaign2)),
			Items:        RewardCapmgaign2,
			Unk02:        0,
		}
		rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeAchievement), BuildAchievementCampaign(0x50, &reward, pkt.CampaignId))
		SendPacket(rst, uPtr.CurrentConnection)
		DebugInfo(1, "User", uPtr.UserName, "get achievement campaign id", pkt.CampaignId)
	case Campaign_3:
		reward := OutAchievementCampaign{
			Unk00:        0,
			Unk01:        0,
			RewardTitle:  0,
			RewardIcon:   0,
			RewardPoints: 0,
			RewardXp:     8000,
			NumOfItems:   uint8(len(RewardCapmgaign3)),
			Items:        RewardCapmgaign3,
			Unk02:        0,
		}
		rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeAchievement), BuildAchievementCampaign(0x60, &reward, pkt.CampaignId))
		SendPacket(rst, uPtr.CurrentConnection)
		DebugInfo(1, "User", uPtr.UserName, "get achievement campaign id", pkt.CampaignId)
	case Campaign_4:
		reward := OutAchievementCampaign{
			Unk00:        0,
			Unk01:        0,
			RewardTitle:  0,
			RewardIcon:   0,
			RewardPoints: 0,
			RewardXp:     10000,
			NumOfItems:   uint8(len(RewardCapmgaign4)),
			Items:        RewardCapmgaign4,
			Unk02:        0,
		}
		rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeAchievement), BuildAchievementCampaign(0x60, &reward, pkt.CampaignId))
		SendPacket(rst, uPtr.CurrentConnection)
		DebugInfo(1, "User", uPtr.UserName, "get achievement campaign id", pkt.CampaignId)
	case Campaign_5:
		reward := OutAchievementCampaign{
			Unk00:        0,
			Unk01:        0,
			RewardTitle:  0,
			RewardIcon:   0,
			RewardPoints: 0,
			RewardXp:     0,
			NumOfItems:   uint8(len(RewardCapmgaign5)),
			Items:        RewardCapmgaign5,
			Unk02:        0,
		}
		rst := BytesCombine(BuildHeader(uPtr.CurrentSequence, PacketTypeAchievement), BuildAchievementCampaign(0x40, &reward, pkt.CampaignId))
		SendPacket(rst, uPtr.CurrentConnection)
		DebugInfo(1, "User", uPtr.UserName, "get achievement campaign id", pkt.CampaignId)
	case Campaign_6:

		DebugInfo(1, "User", uPtr.UserName, "get achievement campaign id", pkt.CampaignId)
	default:
		DebugInfo(2, "Error : Client from", client.RemoteAddr().String(), "sent a unkown AchievementCampaign packet id", pkt.CampaignId)
	}
}

func BuildAchievementCampaign(flags uint32, src *OutAchievementCampaign, id uint16) []byte {
	if src == nil {
		return []byte{}
	}
	buf := make([]byte, 25+src.NumOfItems*8)
	offset := 0
	WriteUint8(&buf, campaign, &offset)
	WriteUint16(&buf, id, &offset)
	WriteUint32(&buf, flags, &offset)

	if flags&0x1 != 0 {
		WriteUint16(&buf, src.Unk00, &offset)
	}
	if flags&0x2 != 0 {
		WriteUint32(&buf, src.Unk01, &offset)
	}
	if flags&0x4 != 0 {
		WriteUint16(&buf, src.RewardTitle, &offset)
	}
	if flags&0x8 != 0 {
		WriteUint16(&buf, src.RewardIcon, &offset)
	}
	if flags&0x10 != 0 {
		WriteUint32(&buf, src.RewardPoints, &offset)
	}
	if flags&0x20 != 0 {
		WriteUint32(&buf, src.RewardXp, &offset)
	}
	if flags&0x40 != 0 {
		WriteUint8(&buf, src.NumOfItems, &offset)
		for _, v := range src.Items {
			WriteUint32(&buf, v.ItemId, &offset)
			WriteUint16(&buf, v.Ammount, &offset)
			WriteUint16(&buf, v.TimeLimited, &offset)
		}
	}
	if flags&0x80 != 0 {
		WriteUint16(&buf, src.Unk02, &offset)
	}
	return buf[:offset]
}

func InitCampaignReward() {
	for _, v := range BoxIDs {
		RewardCapmgaign5 = append(RewardCapmgaign5, OutAchievementCampaignItems{ItemId: v, Ammount: 1, TimeLimited: 0})
	}
}
