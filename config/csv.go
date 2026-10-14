package config

import (
	"bytes"
	"encoding/csv"
	"io"
	"strconv"

	. "github.com/KouKouChan/CSO2-Server/blademaster/typestruct"
	"github.com/KouKouChan/CSO2-Server/builtin"
	. "github.com/KouKouChan/CSO2-Server/logging"
)

const (
	WeaponListCSV  = "scripts/item_list.csv"
	ExpLevelCSV    = "scripts/exp_level.csv"
	UnlockCSV      = "scripts/item_unlock.csv"
	BoxCSV         = "scripts/supplyList.csv"
	VipCSV         = "scripts/vip_info.csv"
	DefaultItemCSV = "scripts/defaultItemList.csv"
	ShopItemCSV    = "scripts/shop.csv"
)

func InitCSV() {
	Info("Reading built-in game data ...")
	readWeaponList()
	readUnlockList()
	readBoxList()
	readDefaultItemList()
	if Conf.EnableShop {
		readShopItemList()
	}
}

func readWeaponList() {
	//读取武器数据
	data, err := builtin.ReadFile(WeaponListCSV)
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if err == nil && len(record[1]) > 16 {
			id, err := strconv.Atoi(record[0])
			if err != nil {
				continue
			}
			itemtype, err := strconv.Atoi(record[12])
			if err != nil {
				continue
			}
			itemd := ItemData{
				ItemID:      uint32(id),
				Name:        record[1][16:],
				Class:       record[4],
				Category:    record[5],
				BuyCategory: record[6],
				ItemType:    itemtype,
			}

			ItemList[itemd.ItemID] = itemd
		} else {
			continue
		}
	}

}

func readUnlockList() {
	//读取武器解锁数据
	data, err := builtin.ReadFile(UnlockCSV)
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if err == nil {
			id, err := strconv.Atoi(record[0])
			if err != nil {
				continue
			}
			nextid, err := strconv.Atoi(record[1])
			if err != nil {
				continue
			}
			flag0, err := strconv.Atoi(record[2])
			if err != nil {
				continue
			}
			count0, err := strconv.Atoi(record[3])
			if err != nil {
				continue
			}
			flag1, err := strconv.Atoi(record[4])
			if err != nil {
				continue
			}
			count1, err := strconv.Atoi(record[5])
			if err != nil {
				continue
			}
			flag2, err := strconv.Atoi(record[6])
			if err != nil {
				continue
			}
			count2, err := strconv.Atoi(record[7])
			if err != nil {
				continue
			}
			flag3, err := strconv.Atoi(record[8])
			if err != nil {
				continue
			}
			count3, err := strconv.Atoi(record[9])
			if err != nil {
				continue
			}
			flag4, err := strconv.Atoi(record[10])
			if err != nil {
				continue
			}
			count4, err := strconv.Atoi(record[11])
			if err != nil {
				continue
			}
			cat, err := strconv.Atoi(record[12])
			if err != nil {
				continue
			}
			itemd := UnlockData{
				ItemID:         uint32(id),
				NextItemID:     uint32(nextid),
				ConditionFlag0: uint32(flag0),
				Count0:         uint32(count0),
				ConditionFlag1: uint32(flag1),
				Count1:         uint32(count1),
				ConditionFlag2: uint32(flag2),
				Count2:         uint32(count2),
				ConditionFlag3: uint32(flag3),
				Count3:         uint32(count3),
				ConditionFlag4: uint32(flag4),
				Count4:         uint32(count4),
				Category:       uint32(cat),
			}

			UnlockList[itemd.NextItemID] = itemd
		} else {
			continue
		}
	}
}

func readBoxList() {
	//读取箱子数据
	data, err := builtin.ReadFile(BoxCSV)
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(bytes.NewReader(data))

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if err == nil {
			boxid, err := strconv.Atoi(record[0])
			if err != nil {
				continue
			}
			boxname := record[1]
			itemid, err := strconv.Atoi(record[2])
			if err != nil {
				continue
			}
			itemname := record[3]
			value, err := strconv.Atoi(record[4])
			if err != nil {
				continue
			}
			keyid, err := strconv.Atoi(record[5])
			if err != nil {
				continue
			}
			day, err := strconv.Atoi(record[6])
			if err != nil {
				continue
			}
			//保存当前物品数据
			if value <= 0 {
				Warn("Illegal value", value, "for item", itemid, "in box", boxid)
				continue
			}
			item := BoxItem{ItemID: uint32(itemid), ItemName: itemname, Value: value, Day: uint16(day)}

			if v, ok := BoxList[uint32(boxid)]; ok {
				//如果该box数据已经存在
				v.Items = append(v.Items, item)
				v.TotalValue += value
				BoxList[uint32(boxid)] = v
			} else {
				BoxList[uint32(boxid)] = BoxData{
					BoxID:      uint32(boxid),
					BoxName:    boxname,
					Items:      []BoxItem{item},
					TotalValue: value,
					KeyId:      uint32(keyid),
					OptId:      0,
				}
				BoxIDs = append(BoxIDs, uint32(boxid))
			}
		} else {
			continue
		}
	}
}

func readDefaultItemList() {
	//读取数据
	data, err := builtin.ReadFile(DefaultItemCSV)
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	DefaultInventoryItem = []UserInventoryItem{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if err == nil {
			itemid, err := strconv.Atoi(record[0])
			if err != nil {
				continue
			}
			count, err := strconv.Atoi(record[2])
			if err != nil {
				continue
			}
			item := UserInventoryItem{Id: uint32(itemid), Count: uint16(count), Type: 1, Time: 0}
			DefaultInventoryItem = append(DefaultInventoryItem, item)
		} else {
			continue
		}
	}
}

func readShopItemList() {
	//读取数据
	data, err := builtin.ReadFile(ShopItemCSV)
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(bytes.NewReader(data))
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if err == nil {
			itemid, err := strconv.Atoi(record[0])
			if err != nil {
				continue
			}
			price, err := strconv.Atoi(record[1])
			if err != nil {
				continue
			}
			currency, err := strconv.Atoi(record[2])
			if err != nil {
				continue
			}
			count, err := strconv.Atoi(record[3])
			if err != nil {
				continue
			}
			day, err := strconv.Atoi(record[4])
			if err != nil {
				continue
			}

			found := false
			for k := range ShopItemList {
				if ShopItemList[k].ItemID == uint32(itemid) {
					ShopItemList[k].Opt = append(ShopItemList[k].Opt, ItemOption{Price: uint32(price), Count: uint16(count), Day: uint16(day)})
					found = true
					break
				}
			}
			if !found {
				ShopItemList = append(ShopItemList, ShopItem{ItemID: uint32(itemid), Currency: uint8(currency), Opt: []ItemOption{{Price: uint32(price), Count: uint16(count), Day: uint16(day)}}})
			}
		} else {
			continue
		}
	}
}
