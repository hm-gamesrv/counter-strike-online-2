package typestruct

type UserLoadout struct {
	Items []uint32
}

func CreateDefaultLoadout() []UserLoadout {
	return []UserLoadout{
		{[]uint32{14, 2, 27, 4, 23, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{[]uint32{3, 15, 27, 4, 23, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{[]uint32{101, 15, 80, 4, 23, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	}
}
