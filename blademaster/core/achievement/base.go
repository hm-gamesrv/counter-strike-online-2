package achievement

import (
	"net"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/logging"
)

const campaign = 3

func OnAchievement(p *PacketData, client net.Conn) {
	var pkt InAchievementPacket
	if p.PraseInAchievementPacket(&pkt) {
		switch pkt.Type {
		case campaign:
			OnAchievementCampaign(p, client)
		default:
			DebugInfo(2, "Unknown achievement packet", pkt.Type, "from", client.RemoteAddr().String())
		}
	} else {
		DebugInfo(2, "Error : Recived a illegal achievement packet from", client.RemoteAddr().String())
	}
}
