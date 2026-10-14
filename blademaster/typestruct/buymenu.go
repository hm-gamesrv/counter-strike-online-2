package typestruct

// UserBuyMenu 用户的购买菜单
type UserBuyMenu struct {
	Pistols     []uint32
	Shotguns    []uint32
	Smgs        []uint32
	Rifles      []uint32
	Snipers     []uint32
	Machineguns []uint32
	Melees      []uint32
	Equipment   []uint32
}

func CreateDefaultUserBuyMenu() UserBuyMenu {
	return UserBuyMenu{
		Pistols:     []uint32{2, 15, 0, 0, 0, 0, 0, 0, 0},
		Shotguns:    []uint32{19, 101, 0, 0, 0, 0, 0, 0, 0},
		Smgs:        []uint32{6, 21, 0, 0, 0, 0, 0, 0, 0},
		Rifles:      []uint32{13, 14, 128, 0, 0, 0, 0, 0, 0},
		Snipers:     []uint32{3, 0, 0, 0, 0, 0, 0, 0, 0},
		Machineguns: []uint32{18, 0, 0, 0, 0, 0, 0, 0, 0},
		Melees:      []uint32{27, 80, 0, 0, 0, 0, 0, 0, 0},
		Equipment:   []uint32{36, 37, 23, 4, 8, 34, 0, 0, 0},
	}
}
