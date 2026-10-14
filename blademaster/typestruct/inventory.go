package typestruct

import (
	. "cso2-server/kerlong"
)

type (
	UserInventory struct {
		NumOfItem            uint16              //物品数量
		Items                []UserInventoryItem //物品
		OnlyOnceBundleItemId map[uint32]bool
		CTModel              uint32 //当前的CT模型
		TModel               uint32 //当前的T模型
		HeadItem             uint32 //当前的头部装饰
		GloveItem            uint32 //当前的手套
		BackItem             uint32 //当前的背部物品
		StepsItem            uint32 //当前的脚步效果
		CardItem             uint32 //当前的卡片
		SprayItem            uint32 //当前的喷漆

		BuyMenu  UserBuyMenu //购买菜单
		Loadouts []UserLoadout
	}
	UserInventoryItem struct {
		Id    uint32 //物品id
		Count uint16 //数量
		Type  uint8
		Time  uint64
	}
)

var (
	DefaultInventoryItem = []UserInventoryItem{}
)

func WriteItem(num uint32, curitem *uint8) []byte {
	buf := make([]byte, 5)
	offset := 0
	WriteUint8(&buf, *curitem, &offset)
	(*curitem)++
	WriteUint32(&buf, num, &offset)
	return buf
}

func CreateNewUserInventory() UserInventory {
	Inv := UserInventory{
		NumOfItem:            0,
		Items:                DefaultInventoryItem,
		OnlyOnceBundleItemId: map[uint32]bool{},
		CTModel:              1001,
		TModel:               1004,
		HeadItem:             0,
		GloveItem:            0,
		BackItem:             0,
		StepsItem:            0,
		CardItem:             0,
		SprayItem:            0,
		BuyMenu:              CreateDefaultUserBuyMenu(),
		Loadouts:             CreateDefaultLoadout(),
	}
	Inv.NumOfItem = uint16(len(Inv.Items))
	return Inv
}
func IsIllegal(num uint32) bool {
	switch num {
	case 2:
		return true
	case 3:
		return true
	case 4:
		return true
	case 6:
		return true
	case 8:
		return true
	case 13:
		return true
	case 14:
		return true
	case 15:
		return true
	case 18:
		return true
	case 19:
		return true
	case 21:
		return true
	case 23:
		return true
	case 27:
		return true
	case 56:
		return true
	case 58:
		return true
	case 69:
		return true
	case 107:
		return true
	case 117:
		return true
	case 134:
		return true
	case 139:
		return true
	case 5172:
		return true
	case 5173:
		return true
	case 5174:
		return true
	case 5227:
		return true
	case 5228:
		return true
	case 5229:
		return true
	default:
		return false
	}
}
