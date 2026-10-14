package typestruct

import (
	"sync"

	. "github.com/KouKouChan/CSO2-Server/kerlong"
	. "github.com/KouKouChan/CSO2-Server/logging"
)

// UManager 全局用户管理
type UManager struct {
	UserNum int
	Users   map[uint32]*User
	Lock    sync.RWMutex
}

var (
	//UsersManager 全局用户管理
	UsersManager = UManager{
		Users: map[uint32]*User{},
	}
	MaxUsers uint32 = 0
)

const (
	//MAXUSERNUM , 8096 is enough
	MAXUSERNUM = 8096
)

func (dest *UManager) AddUser(src *User) bool {
	if dest == nil || src == nil {
		return false
	}
	if src.Userid <= 0 {
		DebugInfo(2, "Usermanager Error : ID of User", (*src).UserName, "is ", src.Userid)
		return false
	}

	// The map and the counter are only consistent while the write lock is held,
	// so the "is it already there?" check has to happen inside it too.
	dest.Lock.Lock()
	defer dest.Lock.Unlock()

	if dest.UserNum > MAXUSERNUM {
		DebugInfo(2, "Usermanager Error : Online users is too more to add user !")
		return false
	}
	if _, ok := dest.Users[src.Userid]; ok {
		DebugInfo(2, "Usermanager Error : User is already in !")
		return false
	}
	dest.UserNum++
	dest.Users[src.Userid] = src
	return true
}

func (dest *UManager) DelUser(src *User) bool {
	if dest == nil || src == nil {
		return false
	}
	if src.Userid <= 0 {
		DebugInfo(2, "Usermanager Error : ID of User", (*src).UserName, "is illegal !")
		return false
	}

	dest.Lock.Lock()
	defer dest.Lock.Unlock()

	if _, ok := dest.Users[src.Userid]; !ok {
		return false
	}
	delete(dest.Users, src.Userid)
	dest.UserNum--
	return true
}

// DelUserByID removes a user by id. Safe to call for unknown ids.
func (dest *UManager) DelUserByID(id uint32) bool {
	if dest == nil {
		return false
	}

	dest.Lock.Lock()
	defer dest.Lock.Unlock()

	if _, ok := dest.Users[id]; !ok {
		return false
	}
	delete(dest.Users, id)
	dest.UserNum--
	return true
}

// SnapshotUsers returns the currently online users.
//
// Callers must iterate over a snapshot rather than over dest.Users directly:
// ranging a map that another goroutine is writing to is a fatal runtime error
// that recover() cannot catch.
func (dest *UManager) SnapshotUsers() []*User {
	if dest == nil {
		return nil
	}

	dest.Lock.RLock()
	defer dest.Lock.RUnlock()

	users := make([]*User, 0, len(dest.Users))
	for _, u := range dest.Users {
		if u != nil {
			users = append(users, u)
		}
	}
	return users
}

// OnlineCount returns the number of users currently online.
func (dest *UManager) OnlineCount() int {
	if dest == nil {
		return 0
	}

	dest.Lock.RLock()
	defer dest.Lock.RUnlock()
	return len(dest.Users)
}

func (dest *UManager) GetChannelUsers(chlsrvid, chlid uint8) []byte {
	if dest == nil {
		return []byte{0, 0}
	}

	buf := make([]byte, 2)
	offset, num := 0, 0

	dest.Lock.RLock()
	defer dest.Lock.RUnlock()

	if len(dest.Users) == 0 {
		DebugInfo(2, "Usermanager Error : There is no online user !")
		return []byte{0, 0}
	}

	for _, u := range dest.Users {
		if u.GetUserChannelServerID() == chlsrvid &&
			u.GetUserChannelID() == chlid {
			num++

			tmpbuf := make([]byte, 128)
			WriteUint32(&tmpbuf, u.Userid, &offset)
			WriteString(&tmpbuf, []byte(u.IngameName), &offset)

			buf = BytesCombine(buf, tmpbuf[:offset], BuildUserInfo(0xFFFFFFFF, NewUserInfo(u), 0, false))

			offset = 0
		}
	}
	WriteUint16(&buf, uint16(num), &offset)
	return buf[:offset]
}
