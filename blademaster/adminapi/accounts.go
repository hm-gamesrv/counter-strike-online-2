package adminapi

import (
	"errors"
	"fmt"
	"os"
	"strings"

	. "github.com/KouKouChan/CSO2-Server/blademaster/typestruct"
	. "github.com/KouKouChan/CSO2-Server/kerlong"
	. "github.com/KouKouChan/CSO2-Server/logging"
	. "github.com/KouKouChan/CSO2-Server/servermanager"
)

// maxAccountNameLength mirrors the limit enforced by the storage layer.
const maxAccountNameLength = 64

var (
	// ErrInvalidAccount means the request would create an inconsistent account.
	ErrInvalidAccount = errors.New("invalid account")
	// ErrNameInUse means the login name is already taken. The storage layer's
	// own sentinels are mapped onto this pair so that the HTTP layer does not
	// have to leak "storage:" prefixed messages.
	ErrNameInUse = errors.New("username is already taken")
	// ErrNicknameInUse means the nickname is already taken.
	ErrNicknameInUse = errors.New("nickname is already taken")
)

// CreateAccountRequest is the body of POST /api/accounts.
type CreateAccountRequest struct {
	UserName string `json:"username"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
	Mail     string `json:"mail"`
	GM       bool   `json:"gm"`
	VIP      bool   `json:"vip"`
}

// AccountPatch is the body of PATCH /api/accounts/{name}.
//
// Every field is a pointer so that "leave this alone" and "set this to zero"
// are distinguishable. A zero-valued patch changes nothing.
type AccountPatch struct {
	Nickname *string `json:"nickname"`
	Mail     *string `json:"mail"`
	Password *string `json:"password"`
	Level    *uint16 `json:"level"`
	Points   *uint64 `json:"points"`
	Cash     *uint32 `json:"cash"`
	Mpoints  *uint32 `json:"mpoints"`
	VipLevel *uint8  `json:"vipLevel"`
	GM       *bool   `json:"gm"`
}

func validateName(field, value string) error {
	if value == "" {
		return fmt.Errorf("%w: %s must not be empty", ErrInvalidAccount, field)
	}
	if len(value) > maxAccountNameLength {
		return fmt.Errorf("%w: %s must be at most %d characters", ErrInvalidAccount, field, maxAccountNameLength)
	}
	return nil
}

// ListAllAccounts returns every account, from the in-memory index.
func ListAllAccounts() []AccountRef {
	return ListAccounts()
}

// CreateAccount registers a new account.
func CreateAccount(req CreateAccountRequest) (*AccountDetail, error) {
	req.UserName = strings.TrimSpace(req.UserName)
	req.Nickname = strings.TrimSpace(req.Nickname)

	if err := validateName("username", req.UserName); err != nil {
		return nil, err
	}
	if err := validateName("nickname", req.Nickname); err != nil {
		return nil, err
	}
	if req.Password == "" {
		return nil, fmt.Errorf("%w: password must not be empty", ErrInvalidAccount)
	}

	hash, err := NewPasswordHash(req.Password)
	if err != nil {
		return nil, err
	}

	u := GetNewUser()
	u.SetUserName(req.UserName, req.Nickname)
	u.Password = []byte(hash)
	u.UserMail = req.Mail
	if req.GM {
		u.SetGM()
	}
	if req.VIP {
		// SetVIP derives the level from Level and GM status, matching what the
		// game itself would compute.
		u.SetVIP()
	}

	// CreateUser claims the names atomically, so a concurrent duplicate cannot
	// slip through between an existence check and the write.
	if err = CreateUser(&u); err != nil {
		switch {
		case errors.Is(err, ErrUsernameTaken):
			return nil, ErrNameInUse
		case errors.Is(err, ErrNicknameTaken):
			return nil, ErrNicknameInUse
		}
		return nil, err
	}

	Info("admin: created account", req.UserName)
	return detailFromUser(&u, false), nil
}

// applyPatch mutates an account in place.
//
// passwordHash is prepared by the caller before the lock is taken, because
// PBKDF2 with 100k iterations is far too slow to run while holding it.
func applyPatch(u *User, patch AccountPatch, passwordHash []byte) {
	u.UserMutex.Lock()
	defer u.UserMutex.Unlock()

	if patch.Nickname != nil {
		u.IngameName = strings.TrimSpace(*patch.Nickname)
	}
	if patch.Mail != nil {
		u.UserMail = *patch.Mail
	}
	if patch.Level != nil {
		u.Level = *patch.Level
	}
	if patch.Points != nil {
		u.Points = *patch.Points
	}
	if patch.Cash != nil {
		u.Cash = *patch.Cash
	}
	if patch.Mpoints != nil {
		u.Mpoints = *patch.Mpoints
	}
	if patch.VipLevel != nil {
		u.VipLevel = *patch.VipLevel
	}
	if patch.GM != nil {
		if *patch.GM {
			u.Gm = 1
		} else {
			u.Gm = 0
		}
	}
	if passwordHash != nil {
		u.Password = passwordHash
	}
}

// UpdateAccount applies a partial update to an account.
//
// If the player is currently connected the live in-memory object is updated, so
// the change is not undone by the next periodic save, and the client is told to
// refresh. Otherwise the stored document is edited directly.
func UpdateAccount(username string, patch AccountPatch) (*AccountDetail, error) {
	u := findOnlineUser(username)
	online := u != nil

	if u == nil {
		loaded, err := LoadUser(username)
		if err != nil {
			return nil, ErrUserNotFound
		}
		u = loaded
	}

	if patch.Nickname != nil {
		newNickname := strings.TrimSpace(*patch.Nickname)
		if err := validateName("nickname", newNickname); err != nil {
			return nil, err
		}
		if newNickname != u.IngameName && NicknameExists(newNickname) {
			return nil, ErrNicknameInUse
		}
	}

	var passwordHash []byte
	if patch.Password != nil {
		if *patch.Password == "" {
			return nil, fmt.Errorf("%w: password must not be empty", ErrInvalidAccount)
		}
		hash, err := NewPasswordHash(*patch.Password)
		if err != nil {
			return nil, err
		}
		passwordHash = []byte(hash)
	}

	applyPatch(u, patch, passwordHash)

	// Write through immediately: an admin edit should be durable, and this also
	// retires a nickname that has just been changed.
	if err := SaveUser(u); err != nil {
		return nil, err
	}

	if online && u.CurrentConnection != nil {
		rst := BytesCombine(BuildHeader(u.CurrentSequence, PacketTypeUserInfo),
			BuildUserInfo(0xFFFFFFFF, NewUserInfo(u), u.Userid, true))
		SendPacket(rst, u.CurrentConnection)
	}

	Info("admin: updated account", username, "(online:", online, ")")
	return detailFromUser(u, online), nil
}

// DeleteAccount removes an account. A connected player is disconnected first so
// that the periodic save cannot write the document back after deletion.
func DeleteAccount(username string) error {
	if findOnlineUser(username) != nil {
		if err := KickUser(username); err != nil {
			return err
		}
	}

	if err := DeleteUser(username); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrUserNotFound
		}
		return err
	}

	Info("admin: deleted account", username)
	return nil
}
