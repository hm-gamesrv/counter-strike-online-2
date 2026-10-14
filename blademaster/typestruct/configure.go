package typestruct

type ItemData struct {
	ItemID   uint32
	Name     string
	Category string
	ItemType int
}

type BoxData struct {
	BoxID      uint32
	BoxName    string
	Items      []BoxItem
	TotalValue int
	KeyId      uint32
	OptId      uint32
}

type BoxItem struct {
	ItemID uint32
	Value  int
	Day    uint16
}

type ShopItem struct {
	ItemID   uint32
	Currency uint8
	Opt      []ItemOption
}

type ItemOption struct {
	Price uint32
	Count uint16
	Day   uint16
}

var (
	ItemList     = make(map[uint32]ItemData)
	BoxList      = make(map[uint32]BoxData)
	BoxIDs       = []uint32{}
	ShopItemList = []ShopItem{}
)
