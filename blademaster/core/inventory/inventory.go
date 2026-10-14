package inventory

import (
	. "cso2-server/blademaster/typestruct"
	. "cso2-server/config"
	. "cso2-server/kerlong"
)

var (
	FullInventoryItem  []UserInventoryItem
	FullInventoryReply []byte
)

func BuildInventoryInfo(u *User) []byte {
	if Conf.UnlockAllWeapons {
		return FullInventoryReply
	}
	buf := make([]byte, 5+u.Inventory.NumOfItem*19)
	offset, offset_front := 2, 0
	num := 0

	for k, v := range u.Inventory.Items {
		if v.Count <= 0 {
			continue
		}
		WriteUint16(&buf, uint16(k), &offset)
		WriteUint8(&buf, 1, &offset)
		WriteUint32(&buf, v.Id, &offset)
		WriteUint16(&buf, v.Count, &offset)
		WriteUint8(&buf, 1, &offset)
		WriteUint8(&buf, 0, &offset)
		WriteUint64(&buf, v.Time, &offset)

		num++
	}
	WriteUint16(&buf, uint16(num), &offset_front) //把实际的数量写进前部
	return buf[:offset]
}

func BuildInventoryInfoSingle(u *User, itemid uint32, idx int) []byte {
	buf := make([]byte, 128)
	offset := 0
	WriteUint16(&buf, 1, &offset)
	if idx < 0 || idx >= len(u.Inventory.Items) {
		for k, v := range u.Inventory.Items {
			if v.Id != itemid {
				continue
			}
			WriteUint16(&buf, uint16(k), &offset)
			if v.Count == 0 {
				WriteUint8(&buf, 0, &offset) //existed
			} else {
				WriteUint8(&buf, 1, &offset)
			}
			WriteUint32(&buf, v.Id, &offset)
			WriteUint16(&buf, v.Count, &offset)
			WriteUint8(&buf, 1, &offset)
			WriteUint8(&buf, 0, &offset)
			WriteUint64(&buf, v.Time, &offset)
			break
		}
	} else {
		WriteUint16(&buf, uint16(idx), &offset)
		if u.Inventory.Items[idx].Count == 0 {
			WriteUint8(&buf, 0, &offset) //existed
		} else {
			WriteUint8(&buf, 1, &offset)
		}
		WriteUint32(&buf, u.Inventory.Items[idx].Id, &offset)
		WriteUint16(&buf, u.Inventory.Items[idx].Count, &offset)
		WriteUint8(&buf, 1, &offset)
		WriteUint8(&buf, 0, &offset)
		WriteUint64(&buf, u.Inventory.Items[idx].Time, &offset)
	}
	return buf[:offset]
}

func BuildFullInventoryInfo() []byte {
	buf := make([]byte, 5+uint16(len(FullInventoryItem))*19)
	offset := 0
	WriteUint16(&buf, uint16(len(FullInventoryItem)), &offset)
	for k, v := range FullInventoryItem {
		WriteUint16(&buf, uint16(k), &offset)
		WriteUint8(&buf, 1, &offset)
		WriteUint32(&buf, v.Id, &offset)
		WriteUint16(&buf, v.Count, &offset)
		WriteUint8(&buf, 1, &offset)
		WriteUint8(&buf, 0, &offset)
		WriteUint64(&buf, v.Time, &offset)
	}
	return buf[:offset]
}

func BuildUnlockReply(u *User) []byte {
	if u == nil {
		return []byte{}
	}
	buf := make([]byte, 4096)
	offset := 0
	WriteUint8(&buf, 1, &offset)                            //type ?
	WriteUint16(&buf, uint16(len(UnlockFullList)), &offset) //num of weapons
	for k, v := range UnlockFullList {
		WriteUint32(&buf, v.Itemid, &offset)
		WriteUint32(&buf, uint32(k), &offset)
		WriteUint8(&buf, v.CostType, &offset)
		WriteUint32(&buf, v.Price, &offset)
	}

	WriteUint16(&buf, 1, &offset) //num of weapons

	WriteUint32(&buf, 2, &offset) //前置
	WriteUint32(&buf, 1, &offset) //当前
	WriteUint32(&buf, 2, &offset) //杀敌数
	WriteUint16(&buf, 1, &offset)
	WriteUint16(&buf, 1, &offset)
	WriteUint16(&buf, 1, &offset)

	WriteUint16(&buf, 1, &offset) //unk

	WriteUint32(&buf, 1, &offset)

	return buf[:offset]
}

func CreateFullInventoryItem() []UserInventoryItem {
	items := []UserInventoryItem{}
	var i uint32
	//用户角色
	for i = 1001; i <= 1058; i++ {
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	//添加默认武器
	number := []uint32{2, 3, 4, 6, 8, 13, 14, 15, 18, 19, 21, 23, 27, 34, 36, 37, 80, 128, 101, 49009, 49004}
	for _, v := range number {
		items = append(items, UserInventoryItem{Id: v, Count: 1, Type: 1, Time: 0})
	}
	//解锁武器
	for i = 1; i <= 33; i++ {
		if IsIllegal(i) {
			continue
		}
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	for i = 44; i <= 163; i++ {
		if IsIllegal(i) {
			continue
		}
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	//隔离区技能
	items = append(items, UserInventoryItem{Id: 2019, Count: 1, Type: 1, Time: 0})
	items = append(items, UserInventoryItem{Id: 2020, Count: 1, Type: 1, Time: 0})
	for i = 2021; i <= 2023; i++ {
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	//武器皮肤
	for i = 5042; i <= 5370; i++ {
		if IsIllegal(i) {
			continue
		}
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	items = append(items, UserInventoryItem{Id: 5997, Count: 1, Type: 1, Time: 0})
	//帽子
	for i = 10001; i <= 10133; i++ {
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	//背包
	for i = 20001; i <= 20107; i++ {
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	//手套
	for i = 30001; i <= 30027; i++ {
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	//脚部特效
	for i = 40001; i <= 40025; i++ {
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	//喷漆
	for i = 42001; i <= 42020; i++ {
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	//道具
	for i = 49001; i <= 49010; i++ {
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	items = append(items, UserInventoryItem{Id: 49999, Count: 1, Type: 1, Time: 0})
	//角色卡片
	for i = 60001; i <= 60004; i++ {
		items = append(items, UserInventoryItem{Id: i, Count: 1, Type: 1, Time: 0})
	}
	return items
}
