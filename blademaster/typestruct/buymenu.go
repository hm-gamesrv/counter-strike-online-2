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

func CreateFullUserBuyMenu() UserBuyMenu {
	return UserBuyMenu{
		Pistols:     []uint32{5280, 5279, 5337, 5356, 5294, 5360, 5262, 103, 106},
		Shotguns:    []uint32{5130, 5293, 5306, 5261, 5242, 5264, 5265, 5230, 137},
		Smgs:        []uint32{5251, 5295, 5238, 5320, 5285, 5347, 5310, 162, 105},
		Rifles:      []uint32{46, 45, 5296, 5184, 5355, 113, 102, 161, 157},
		Snipers:     []uint32{5133, 5118, 5206, 5241, 5225, 146, 125, 160, 163},
		Machineguns: []uint32{5125, 5314, 5260, 87, 5332, 5366, 5276, 5233, 159},
		Melees:      []uint32{79, 5232, 84, 5221, 5304, 5330, 5253, 5231, 5353},
		Equipment:   []uint32{36, 37, 23, 4, 8, 34, 0, 0, 0},
	}
}
