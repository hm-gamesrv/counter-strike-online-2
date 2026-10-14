// Package adminapi is the server's administration interface.
//
// It replaces the old plain-text GM console (a bespoke TCP protocol served from
// this same port). Administration is now an HTTP JSON API plus a small web page,
// authenticated with a bearer token taken from the ADMIN_TOKEN environment
// variable.
package adminapi

import (
	"errors"
	"fmt"

	. "github.com/KouKouChan/CSO2-Server/blademaster/core/inventory"
	. "github.com/KouKouChan/CSO2-Server/blademaster/core/message"
	. "github.com/KouKouChan/CSO2-Server/blademaster/core/room"
	. "github.com/KouKouChan/CSO2-Server/blademaster/typestruct"
	. "github.com/KouKouChan/CSO2-Server/kerlong"
	. "github.com/KouKouChan/CSO2-Server/logging"
	. "github.com/KouKouChan/CSO2-Server/servermanager"
)

var (
	// ErrUserNotFound means no account with that name exists.
	ErrUserNotFound = errors.New("user not found")
	// ErrNotOnline means the operation requires the player to be connected.
	ErrNotOnline = errors.New("user is not online")
	// ErrRoomNotFound means the room could not be located in that channel.
	ErrRoomNotFound = errors.New("room not found")
)

// OnlineUser is the admin view of one connected player.
type OnlineUser struct {
	UserName   string `json:"username"`
	IngameName string `json:"nickname"`
	RuntimeID  uint32 `json:"runtimeId"`
	Level      uint16 `json:"level"`
	GM         bool   `json:"gm"`
	VIP        bool   `json:"vip"`
	ChannelSrv uint8  `json:"channelServerIndex"`
	Channel    uint8  `json:"channelIndex"`
	RoomID     uint16 `json:"roomId"`
	InGame     bool   `json:"inGame"`
}

// AccountDetail is the admin view of a single account, online or not.
type AccountDetail struct {
	Uuid          string `json:"uuid"`
	UserName      string `json:"username"`
	IngameName    string `json:"nickname"`
	Mail          string `json:"mail"`
	Level         uint16 `json:"level"`
	Points        uint64 `json:"points"`
	Cash          uint32 `json:"cash"`
	Mpoints       uint32 `json:"mpoints"`
	VIPLevel      uint8  `json:"vipLevel"`
	GM            bool   `json:"gm"`
	PlayedMatches uint32 `json:"playedMatches"`
	Wins          uint32 `json:"wins"`
	Kills         uint32 `json:"kills"`
	Deaths        uint32 `json:"deaths"`
	Assists       uint32 `json:"assists"`
	Headshots     uint32 `json:"headshots"`
	ItemCount     int    `json:"itemCount"`
	Online        bool   `json:"online"`
}

// findOnlineUser returns the connected player with that login name, or nil.
func findOnlineUser(username string) *User {
	for _, u := range UsersManager.SnapshotUsers() {
		if u != nil && u.UserName == username {
			return u
		}
	}
	return nil
}

func detailFromUser(u *User, online bool) *AccountDetail {
	return &AccountDetail{
		Uuid:          u.Uuid,
		UserName:      u.UserName,
		IngameName:    u.IngameName,
		Mail:          u.UserMail,
		Level:         u.Level,
		Points:        u.Points,
		Cash:          u.Cash,
		Mpoints:       u.Mpoints,
		VIPLevel:      u.VipLevel,
		GM:            u.IsGM(),
		PlayedMatches: u.PlayedMatches,
		Wins:          u.Wins,
		Kills:         u.Kills,
		Deaths:        u.Deaths,
		Assists:       u.Assists,
		Headshots:     u.Headshots,
		ItemCount:     len(u.Inventory.Items),
		Online:        online,
	}
}

// ListOnlineUsers returns every connected player.
func ListOnlineUsers() []OnlineUser {
	users := UsersManager.SnapshotUsers()
	list := make([]OnlineUser, 0, len(users))

	for _, u := range users {
		list = append(list, OnlineUser{
			UserName:   u.UserName,
			IngameName: u.IngameName,
			RuntimeID:  u.Userid,
			Level:      u.Level,
			GM:         u.IsGM(),
			VIP:        u.IsVIP(),
			ChannelSrv: u.GetUserChannelServerID(),
			Channel:    u.GetUserChannelID(),
			RoomID:     u.GetUserRoomID(),
			InGame:     u.CurrentIsIngame,
		})
	}
	return list
}

// OnlineUserCount returns how many players are connected.
func OnlineUserCount() int {
	return UsersManager.OnlineCount()
}

// GetAccount returns one account, preferring the live in-memory object when the
// player is connected and falling back to the stored document otherwise.
func GetAccount(username string) (*AccountDetail, error) {
	if u := findOnlineUser(username); u != nil {
		return detailFromUser(u, true), nil
	}

	u, err := LoadUser(username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return detailFromUser(u, false), nil
}

// KickUser disconnects a player.
func KickUser(username string) error {
	u := findOnlineUser(username)
	if u == nil {
		return ErrNotOnline
	}
	if u.CurrentConnection == nil {
		return ErrNotOnline
	}

	OnSendMessage(u.CurrentSequence, u.CurrentConnection, MessageDialogBox, GAME_SERVER_ERROR)
	OnLeaveRoom(u.CurrentConnection, true)
	DelUserWithConn(u.CurrentConnection)
	u.CurrentConnection.Close()

	Info("admin: kicked player", username)
	return nil
}

// withOfflineUser loads an account that is not currently connected, applies fn
// and writes it back.
func withOfflineUser(username string, fn func(u *User)) error {
	u, err := LoadUser(username)
	if err != nil {
		return ErrUserNotFound
	}

	fn(u)
	return SaveUser(u)
}

// AddItem gives an item to a player, whether they are connected or not.
func AddItem(username string, itemID uint32, count uint16) error {
	if u := findOnlineUser(username); u != nil {
		idx := u.AddItem(itemID, count, 0)

		rst := BytesCombine(BuildHeader(u.CurrentSequence, PacketTypeInventory_Create),
			BuildInventoryInfoSingle(u, itemID, idx))
		SendPacket(rst, u.CurrentConnection)
		OnSendMessage(u.CurrentSequence, u.CurrentConnection, MessageNotice, GAME_USER_NEW_ITEM)

		Info("admin: gave item", itemID, "x", count, "to", username)
		return nil
	}

	if err := withOfflineUser(username, func(u *User) { u.AddItem(itemID, count, 0) }); err != nil {
		return err
	}
	Info("admin: gave item", itemID, "x", count, "to offline account", username)
	return nil
}

// RemoveItem takes an item away from a player.
func RemoveItem(username string, itemID uint32) error {
	if u := findOnlineUser(username); u != nil {
		u.RemoveItem(itemID)

		rst := BytesCombine(BuildHeader(u.CurrentSequence, PacketTypeUserInfo),
			BuildUserInfo(0xFFFFFFFF, NewUserInfo(u), u.Userid, true))
		SendPacket(rst, u.CurrentConnection)

		Info("admin: removed item", itemID, "from", username)
		return nil
	}

	if err := withOfflineUser(username, func(u *User) { u.RemoveItem(itemID) }); err != nil {
		return err
	}
	Info("admin: removed item", itemID, "from offline account", username)
	return nil
}

// SetVIP grants VIP status.
func SetVIP(username string) error {
	if u := findOnlineUser(username); u != nil {
		u.SetVIP()

		rst := BytesCombine(BuildHeader(u.CurrentSequence, PacketTypeUserInfo),
			BuildUserInfo(0xFFFFFFFF, NewUserInfo(u), u.Userid, true))
		SendPacket(rst, u.CurrentConnection)

		Info("admin: granted VIP to", username)
		return nil
	}

	if err := withOfflineUser(username, func(u *User) { u.SetVIP() }); err != nil {
		return err
	}
	Info("admin: granted VIP to offline account", username)
	return nil
}

// SetGM grants game-master status.
func SetGM(username string) error {
	if u := findOnlineUser(username); u != nil {
		u.SetGM()

		rst := BytesCombine(BuildHeader(u.CurrentSequence, PacketTypeUserInfo),
			BuildUserInfo(0xFFFFFFFF, NewUserInfo(u), u.Userid, true))
		SendPacket(rst, u.CurrentConnection)

		Info("admin: granted GM to", username)
		return nil
	}

	if err := withOfflineUser(username, func(u *User) { u.SetGM() }); err != nil {
		return err
	}
	Info("admin: granted GM to offline account", username)
	return nil
}

// DeleteRoom force-closes a room.
//
// Note the argument order matches the old `delroom <serveridx> <channelidx>
// <roomid>` console command; DelChannelRoom takes them the other way round.
func DeleteRoom(channelServerID, channelID uint8, roomID uint16) error {
	if !DelChannelRoom(roomID, channelID, channelServerID) {
		return fmt.Errorf("%w: room %d in channel server %d channel %d",
			ErrRoomNotFound, roomID, channelServerID, channelID)
	}

	Info("admin: force-closed room", roomID, "in channel server", channelServerID, "channel", channelID)
	return nil
}

// SaveAll flushes every online account to disk.
func SaveAll() error {
	if !SaveAllUsers() {
		return errors.New("saving one or more accounts failed")
	}
	Info("admin: saved all online accounts")
	return nil
}
