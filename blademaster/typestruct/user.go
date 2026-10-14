package typestruct

import (
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	. "cso2-server/blademaster/Exp"
)

type (
	User struct {
		//个人信息
		CheckUpdate int
		// Uuid is the account's permanent identity and its storage key: the
		// account document is "$DATA_DIR/users/<Uuid>.json". It never changes.
		Uuid string
		// Userid is a runtime-only slot number (1..MAXUSERNUM) handed out on
		// each login. It is not persisted and is unrelated to Uuid.
		Userid               uint32 `json:"-"`
		NexonID              uint64
		UserName             string
		IngameName           string
		Password             []byte
		Gm                   uint8
		Level                uint16
		Rank                 uint8
		RankFrame            uint8
		Points               uint64
		CurrentExp           uint64
		MaxExp               uint64
		PlayedMatches        uint32
		Wins                 uint32
		Kills                uint32
		Headshots            uint32
		Deaths               uint32
		Assists              uint32
		Accuracy             uint16
		SecondsPlayed        uint32
		NetCafeName          []byte
		Cash                 uint32
		ClanName             []byte
		ClanMark             uint32
		WorldRank            uint32
		Campaign             uint16
		Mpoints              uint32
		TitleId              uint16
		UnlockedTitles       []byte
		Signature            []byte
		UnreadedMsg          uint8
		BestGamemode         uint32
		BestMap              uint32
		UnlockedAchievements []byte
		Avatar               uint16
		UnlockedAvatars      []byte
		VipLevel             uint8
		VipXp                uint32
		SkillHumanCurXp      uint64
		SkillHumanMaxXp      uint64
		SkillHumanPoints     uint8
		SkillZombieCurXp     uint64
		SkillZombieMaxXp     uint64
		SkillZombiePoints    uint8
		UserMail             string
		//连接
		CurrentConnection net.Conn `json:"-"`
		//频道房间信息
		CurrentChannelServerIndex uint8       `json:"-"`
		CurrentChannelIndex       uint8       `json:"-"`
		CurrentRoomId             uint16      `json:"-"`
		CurrentTeam               uint8       `json:"-"`
		Currentstatus             uint8       `json:"-"`
		CurrentIsIngame           bool        `json:"-"`
		CurrentSequence           *uint8      `json:"-"`
		CurrentKillNum            uint16      `json:"-"`
		CurrentDeathNum           uint16      `json:"-"`
		CurrentAssistNum          uint16      `json:"-"`
		NetInfo                   UserNetInfo `json:"-"`
		//仓库信息
		Inventory   UserInventory
		WeaponKills map[uint32]uint32

		UserMutex *sync.Mutex `json:"-"`
	}

	UserNetInfo struct {
		ExternalIpAddress  uint32
		ExternalClientPort uint16
		ExternalServerPort uint16
		ExternalTvPort     uint16

		LocalIpAddress  uint32
		LocalClientPort uint16
		LocalServerPort uint16
		LocalTvPort     uint16
	}
)

const (
	//user status
	UserNotReady = 0
	UserIngame   = 1
	UserReady    = 2

	//阵营
	UserForceUnknown          = 0
	UserForceTerrorist        = 1
	UserForceCounterTerrorist = 2

	perDay = float64(15) / 64
)

var (
	ti, _           = time.Parse("2006-01-02 15:04:05", "1970-01-01 00:00:00")
	maxTime, _      = time.Parse("2006-01-02 15:04:05", "2200-01-01 00:00:00")
	maxTimeDuration = uint64(maxTime.Sub(ti).Minutes() * perDay)
)

func (u User) IsVIP() bool {
	if u.VipLevel <= 0 {
		return false
	}
	return true
}

func (u *User) SetVIP() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	//u.VipLevel = 1
	u.checkVIP()
}

func (u *User) checkVIP() {
	if u == nil {
		return
	}
	if u.Gm == 1 {
		u.VipLevel = 6
		return
	}
	u.VipLevel = uint8((u.Level + 4) / 5)
	if u.VipLevel > 5 {
		u.VipLevel = 5
	}
}

func (u *User) SetGM() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.Gm = 1
}

func (u *User) SetID(id uint32) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.Userid = id
}

func (u *User) SetUserName(loginName, username string) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.UserName = loginName
	u.IngameName = username
}

func (u *User) SetUserChannelServer(id uint8) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentChannelServerIndex = id
}

func (u *User) SetUserChannel(id uint8) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentChannelIndex = id
}

func (u *User) QuitChannel() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentChannelIndex = 0
}

func (u *User) QuitRoom() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentRoomId = 0
	u.CurrentTeam = UserForceUnknown
	u.Currentstatus = UserNotReady
	u.CurrentIsIngame = false
}

func (u *User) JoinRoom(id uint16, team uint8) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentRoomId = id
	u.CurrentTeam = team
	u.Currentstatus = UserNotReady
	u.CurrentIsIngame = false
}

func (u *User) SetUserTeam(team uint8) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentTeam = team
}

func (u *User) SetUserStatus(status uint8) {
	if u == nil {
		return
	}
	if status <= 2 &&
		status >= 0 {
		u.UserMutex.Lock()
		defer u.UserMutex.Unlock()
		u.Currentstatus = status
	}
}

// GetUserChannelServerID 获取用户所在分区服务器ID
func (u User) GetUserChannelServerID() uint8 {
	if u.Userid <= 0 {
		return 0
	}
	return u.CurrentChannelServerIndex
}

// 获取用户所在频道ID
func (u User) GetUserChannelID() uint8 {
	if u.Userid <= 0 {
		return 0
	}
	return u.CurrentChannelIndex
}

// 获取用户所在房间ID
func (u User) GetUserRoomID() uint16 {
	if u.Userid <= 0 {
		return 0
	}
	return u.CurrentRoomId
}

func (u User) GetUserTeam() uint8 {
	return u.CurrentTeam
}

func (u User) IsUserReady() bool {
	return u.Currentstatus != UserNotReady
}

func (u *User) SetUserIngame(ingame bool) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentIsIngame = ingame
	if ingame {
		u.Currentstatus = UserIngame
	} else {
		u.Currentstatus = UserNotReady
	}

}

func (u *User) UpdateHolepunch(portId uint16, localPort uint16, externalPort uint16) uint16 {
	if u == nil {
		return 0xFFFF
	}
	switch portId {
	case UDPTypeClient:
		u.UserMutex.Lock()
		defer u.UserMutex.Unlock()
		u.NetInfo.LocalClientPort = localPort
		u.NetInfo.ExternalClientPort = externalPort
		return 0
	case UDPTypeServer:
		u.UserMutex.Lock()
		defer u.UserMutex.Unlock()
		u.NetInfo.LocalServerPort = localPort
		u.NetInfo.ExternalServerPort = externalPort
		return 1
	case UDPTypeSourceTV:
		u.UserMutex.Lock()
		defer u.UserMutex.Unlock()
		u.NetInfo.LocalTvPort = localPort
		u.NetInfo.ExternalTvPort = externalPort
		return 2
	default:
		return 0xFFFF
	}
}

func (u *User) CountKillNum(num uint16) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentKillNum = num
}

func (u *User) CountDeadNum(num uint16) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentDeathNum = num
}
func (u *User) CountAssistNum() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentAssistNum++
}
func (u *User) ResetAssistNum() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentAssistNum = 0
}
func (u *User) ResetKillNum() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.CurrentKillNum = 0
}

func (u *User) ResetDeadNum() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	(*u).CurrentDeathNum = 0
}

func (u *User) SetSignature(sig []byte) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	(*u).Signature = sig
}

func (u *User) SetAvatar(id uint16) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.Avatar = id
}

func (u *User) SetTitle(id uint16) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.TitleId = id
}

func (u *User) UpdateCampaign(id uint16) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.Campaign = u.Campaign | id
}

func (u *User) CheckCampaign(id uint16) bool {
	if u == nil {
		return false
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if u.Campaign&id != 0 {
		return true
	}
	return false
}

func (u *User) SetBuyMenu(menu UserBuyMenu) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.Inventory.BuyMenu = menu
}

func GetNewUser() User {
	var mutex sync.Mutex
	// NOTE: this is an unkeyed literal, so the values line up with the field
	// order of User exactly. Adding a field to User means inserting a value
	// here as well; TestGetNewUserDefaults exists to catch it if you forget.
	return User{CheckUpdate: 0, Uuid: "", Userid: //uuid
	0, NexonID:                                   1, UserName: //nexonid
	"", IngameName://loginname
	"", Password://username,looks can change it to another name
	[]byte{}, Gm://passwd
	0, Level://Gm
	1, Rank://level
	0, RankFrame://rank
	0, Points://rankframe
	10000, CurrentExp://points
	0, MaxExp://curEXP
	LevelExp[0], PlayedMatches://maxEXP
	0, Wins://playermatchs
	0, Kills://wins
	0, Headshots://kills
	0, Deaths://headshots
	0, Assists://deaths
	0, Accuracy://assists
	0, SecondsPlayed:// accuracy
	0, NetCafeName:// secondsPlayed
	NewNullString(), Cash:// netCafeName
	0, ClanName:// cash
	NewNullString(), ClanMark:// clanName
	0, WorldRank:// clanMark
	0, Campaign:// worldRank
	0, Mpoints://campaign
	0, TitleId:// mpoints
	0, UnlockedTitles: // titleId
	[]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, Signature:// unlockedTitles
	NewNullString(), UnreadedMsg:// signature
	0, BestGamemode:// unreadedmsg
	0, BestMap:// bestGamemode
	0, UnlockedAchievements: // bestMap
	[]uint8{0x00, 0x00, 0x18, 0x08, 0x00, 0x00, 0x00, 0x00, 0x42, 0x02,
		0x18, 0xC0, 0x09, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0xC0, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xC8, 0xB7, 0x08, 0x00, 0x00, 0x04, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, Avatar:// unlockedAchievements
	1006, UnlockedAvatars: // avatar
	[]uint8{0x00, 0x00, 0x18, 0x08, 0x00, 0x00, 0x00, 0x00, 0x42, 0x02,
		0x18, 0xC0, 0x09, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0xC0, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0xC8, 0xB7, 0x08, 0x00, 0x00, 0x04, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x3E, 0x00, 0x00}, VipLevel:// unlockedAvatars
	0, VipXp://viplevel
	0, SkillHumanCurXp://vipXp
	0, SkillHumanMaxXp://skillHumanCurXp
	0x19AC, SkillHumanPoints://skillHumanMaxXp
	0, SkillZombieCurXp://skillHumanPoints
	0, SkillZombieMaxXp://skillZombieCurXp
	0x16F6, SkillZombiePoints://skillZombieMaxXp
	0, UserMail://skillZombiePoints
	"", CurrentConnection://mail
	nil, CurrentChannelServerIndex://connection
	1, CurrentChannelIndex://serverid
	0, CurrentRoomId://channelid
	0, CurrentTeam://roomid
	0, Currentstatus://currentTeam
	0, CurrentIsIngame://currentstatus
	false, CurrentSequence://currentIsIngame
	nil, CurrentKillNum://sequence
	0, CurrentDeathNum://currentkillNum
	0, CurrentAssistNum://currentAssists
	0, NetInfo: //currentdeathnum
	UserNetInfo{
		ExternalIpAddress:  0,
		ExternalClientPort: 0,
		ExternalServerPort: 0,
		ExternalTvPort:     0,
		LocalIpAddress:     0,
		LocalClientPort:    0,
		LocalServerPort:    0,
		LocalTvPort:        0,
	}, Inventory:                   CreateNewUserInventory(), WeaponKills: //仓库
	map[uint32]uint32{}, UserMutex: &mutex,
	}
}

func (u *User) GetPoints(num uint64) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if math.MaxUint64-num < u.Points {
		u.Points = math.MaxUint64
	} else {
		u.Points += num
	}
}

func (u *User) UsePoints(num uint64) bool {
	if u == nil {
		return false
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if u.Points < num {
		return false
	}
	u.Points -= num
	return true

}

func (u *User) UseCredits(num uint32) bool {
	if u == nil {
		return false
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if u.Cash < num {
		return false
	}
	u.Cash -= num
	return true

}

func (u *User) UseMPoints(num uint32) bool {
	if u == nil {
		return false
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if u.Mpoints < num {
		return false
	}
	u.Mpoints -= num
	return true
}

func (u *User) GetExp(num uint64) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	for num > 0 {
		if u.MaxExp-u.CurrentExp <= num { //能升级
			num -= u.MaxExp - u.CurrentExp
			u.CurrentExp = 0
			u.Level++
			if u.Level >= MAXLEVEL {
				u.Level = MAXLEVEL
				num = 0
			} else {
				u.MaxExp = LevelExp[u.Level-1]
			}
		} else if u.Level < MAXLEVEL { //不能升级
			u.CurrentExp += num
			num = 0
		} else { //已经满级
			num = 0
		}
	}
	if u.VipLevel > 0 {
		u.checkVIP()
	}
}

func (u *User) GetKills(num uint32) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if math.MaxUint32-u.Kills <= num { //能升级
		u.Kills = math.MaxUint32
	} else { //不能升级
		u.Kills += num
	}
}

func (u *User) GetDeathes(num uint32) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if math.MaxUint32-u.Deaths <= num { //能升级
		u.Deaths = math.MaxUint32
	} else { //不能升级
		u.Deaths += num
	}
}

func (u *User) GetAssists(num uint32) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if math.MaxUint32-u.Assists <= num { //能升级
		u.Assists = math.MaxUint32
	} else { //不能升级
		u.Assists += num
	}
}

func (u *User) AddMatches() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if u.PlayedMatches < math.MaxUint32 {
		u.PlayedMatches++
	}
}

func (u *User) AddWins() {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if u.Wins < math.MaxUint32 {
		u.Wins++
	}
}

func (u *User) CountWeaponKill(itemid uint32) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if _, ok := u.WeaponKills[itemid]; ok {
		if u.WeaponKills[itemid] < math.MaxUint32 {
			u.WeaponKills[itemid]++
		}
	} else {
		u.WeaponKills[itemid] = 1
	}
}

func (u *User) AddItem(itemid uint32, count uint16, times uint64) int {
	if u == nil || itemid == 0 {
		return 0
	}

	if times != 0 {
		return u.ExtendItemTime(itemid, count, times)
	}

	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if IsItemRepetable(itemid) {
		for k, v := range u.Inventory.Items {
			if v.Id == itemid && v.Count <= math.MaxUint16-count {
				u.Inventory.Items[k].Count += count
				return k
			}
		}
	}
	if u.Inventory.NumOfItem < math.MaxUint16 {
		u.Inventory.NumOfItem++
		u.Inventory.Items = append(u.Inventory.Items, UserInventoryItem{Id: itemid, Count: count, Type: 1, Time: times})
		return len(u.Inventory.Items) - 1
	}

	return 0
}

func (u *User) ExtendItemTime(itemid uint32, count uint16, times uint64) int {
	d, _ := time.ParseDuration(fmt.Sprintf("%dh", times*24))

	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	//只叠加有限期的
	for k, v := range u.Inventory.Items {
		if v.Id == itemid && v.Time != 0 {
			u.Inventory.Items[k].Time += uint64(d.Minutes() * perDay)
			return k
		}
	}
	//只有无限期的或者没有该物品
	dd := time.Now().Add(d).Sub(ti)
	if u.Inventory.NumOfItem < math.MaxUint16 {
		u.Inventory.NumOfItem++
		u.Inventory.Items = append(u.Inventory.Items, UserInventoryItem{Id: itemid, Count: count, Type: 1, Time: uint64(dd.Minutes() * perDay)})
		return len(u.Inventory.Items) - 1
	}

	return 0
}

func (u *User) SetBoughtItem(id uint32) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.Inventory.OnlyOnceBundleItemId[id] = true
}

func IsItemRepetable(itemid uint32) bool {
	if ItemList[itemid].ItemType == 1 {
		return true
	}
	return false
}

func (u *User) DecreaseItem(itemid uint32) (int, bool) {
	if u == nil {
		return 0, false
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	for k, v := range u.Inventory.Items {
		if v.Id == itemid && v.Count > 0 {
			u.Inventory.Items[k].Count--
			return k, true
		}
	}
	return 0, false
}

func (u *User) GetItemCount(Idx int) int {
	if u == nil {
		return 0
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if Idx >= 0 && Idx < len(u.Inventory.Items) {
		return int(u.Inventory.Items[Idx].Count)
	}
	return 0
}

func (u *User) UseBox(itemid, keyid uint32) bool {
	if u == nil || itemid == keyid || itemid == 0 {
		return false
	}
	boxused, keyused := false, false
	boxidx, keyidx := 0, 0
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	for k, v := range u.Inventory.Items {
		if v.Id == itemid && v.Count > 0 {
			u.Inventory.Items[k].Count--
			boxused = true
			boxidx = k
		} else if v.Id == keyid && v.Count > 0 {
			u.Inventory.Items[k].Count--
			keyused = true
			keyidx = k
		}
		if (boxused && keyused) || (boxused && keyid == 0) {
			return true
		}
	}
	if boxused {
		u.Inventory.Items[boxidx].Count++
	}
	if keyused {
		u.Inventory.Items[keyidx].Count++
	}
	return false
}

func (u *User) GetItemIDBySeq(seq uint16) uint32 {
	if u == nil {
		return 0
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if len(u.Inventory.Items) < int(seq)+1 {
		return 0
	}
	return u.Inventory.Items[seq].Id
}

func (u *User) RemoveItem(itemid uint32) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	for k, v := range u.Inventory.Items {
		if v.Id == itemid {
			u.Inventory.Items[k].Count = 0
			return
		}
	}
}

func (u *User) SetInventoryItems(items *[]UserInventoryItem) {
	if u == nil {
		return
	}
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	u.Inventory.Items = *items
	u.Inventory.NumOfItem = uint16(len(u.Inventory.Items))
}

func (u User) IsGM() bool {
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	if u.Gm == 1 {
		return true
	}
	return false
}

func (u *User) CheckOutdatedItem() {
	if u == nil {
		return
	}
	dd := time.Now().Sub(ti)
	times := uint64(dd.Minutes() * perDay)
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	for k := 0; k < len(u.Inventory.Items); k++ {
		if (u.Inventory.Items[k].Time != 0 && //时间不为0
			(times > u.Inventory.Items[k].Time || u.Inventory.Items[k].Time > maxTimeDuration)) || //时间过期或过长
			u.Inventory.Items[k].Count == 0 { //数量为0
			u.Inventory.Items = append(u.Inventory.Items[:k], u.Inventory.Items[k+1:]...)
			k-- //回退
		}
	}
}

func (u *User) CheckOutdatedItemIngame() []int {
	var rst []int
	if u == nil {
		return rst
	}
	dd := time.Now().Sub(ti)
	times := uint64(dd.Minutes() * perDay)
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()
	for k := 0; k < len(u.Inventory.Items); k++ {
		if u.Inventory.Items[k].Time != 0 && times > u.Inventory.Items[k].Time {
			u.Inventory.Items[k].Count = 0
			u.Inventory.Items[k].Time = 0
			rst = append(rst, k)
		}
	}
	return rst
}
