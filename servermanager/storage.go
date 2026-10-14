package servermanager

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
)

// Persistent storage: one JSON document per account, named after the account's
// UUID.
//
// This server is meant for a small, trusted deployment, so the whole storage
// story is a directory of files:
//
//	$DATA_DIR/users/<uuid>.json     the account document
//	$DATA_DIR/reports/<name>.txt    abuse reports
//
// Account names are deliberately *not* used as file names: a name can be
// renamed and can contain characters that are awkward or dangerous in a path.
// Instead every account gets a random UUID at creation time, and the mapping
// from login name / nickname to UUID is held in memory, rebuilt from the files
// at startup.
//
// There is no database and no cache layer.
var (
	// DataDir is the root of all persistent data (the DATA_DIR environment
	// variable).
	DataDir string
	// UserDataDir holds one JSON file per account, named "<uuid>.json".
	UserDataDir string
	// ReportPath is where the report package appends abuse reports.
	ReportPath string

	// Dblock serialises account-file reads and writes.
	Dblock sync.Mutex
)

// maxNameLength is a policy limit on account and nickname length.
const maxNameLength = 64

var (
	// ErrInvalidUUID is returned when a storage key is not a canonical UUID.
	ErrInvalidUUID = errors.New("storage: invalid account uuid")
	// ErrUsernameTaken is returned when the login name is already in use.
	ErrUsernameTaken = errors.New("storage: username is already taken")
	// ErrNicknameTaken is returned when the nickname is already in use.
	ErrNicknameTaken = errors.New("storage: nickname is already taken")
)

// indexEntry remembers the names an account was last indexed under, so that a
// rename can retire the old keys.
type indexEntry struct {
	Username string
	Nickname string
}

// The in-memory name index. Account lookups go through these maps rather than
// touching the file system, which also makes name uniqueness enforceable.
var (
	userIndexMu sync.RWMutex
	userByName  = map[string]string{} // login name  -> uuid
	userByNick  = map[string]string{} // nickname    -> uuid
	userByUUID  = map[string]indexEntry{}
)

// InitDataDir points the storage layer at root, creates the directories and
// builds the account index from what is already on disk.
func InitDataDir(root string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	DataDir = abs
	UserDataDir = filepath.Join(abs, "users")
	ReportPath = filepath.Join(abs, "reports")

	for _, dir := range []string{DataDir, UserDataDir, ReportPath} {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return RebuildUserIndex()
}

// RebuildUserIndex re-reads every account document and rebuilds the in-memory
// name index. It runs once at startup.
func RebuildUserIndex() error {
	entries, err := os.ReadDir(UserDataDir)
	if err != nil {
		return err
	}

	byName := make(map[string]string, len(entries))
	byNick := make(map[string]string, len(entries))
	byUUID := make(map[string]indexEntry, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		uuid := strings.TrimSuffix(entry.Name(), ".json")
		u, err := LoadUserByUUID(uuid)
		if err != nil {
			Warn("storage: skipping unreadable account file", entry.Name()+":", err)
			continue
		}
		if u.UserName == "" || u.IngameName == "" {
			Warn("storage: skipping account file with no name", entry.Name())
			continue
		}

		if other, dup := byName[u.UserName]; dup {
			Warn("storage: duplicate login name", u.UserName, "in", entry.Name(), "and", other)
		}
		if other, dup := byNick[u.IngameName]; dup {
			Warn("storage: duplicate nickname", u.IngameName, "in", entry.Name(), "and", other)
		}

		byName[u.UserName] = uuid
		byNick[u.IngameName] = uuid
		byUUID[uuid] = indexEntry{Username: u.UserName, Nickname: u.IngameName}
	}

	userIndexMu.Lock()
	userByName, userByNick, userByUUID = byName, byNick, byUUID
	userIndexMu.Unlock()

	// Diagnostic rather than an event: shown at DebugLevel 1 and above.
	DebugInfo(1, "Loaded", len(byUUID), "account(s) from", UserDataDir)
	return nil
}

// userFilePath validates the UUID and returns the account document's path.
func userFilePath(uuid string) (string, error) {
	if !IsUUID(uuid) {
		return "", ErrInvalidUUID
	}
	return filepath.Join(UserDataDir, uuid+".json"), nil
}

// writeFileAtomic writes data to path via a temporary file and a rename, so a
// crash mid-write cannot leave a half-written account document behind.
func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// UserExists reports whether a login name is taken.
func UserExists(username string) bool {
	userIndexMu.RLock()
	defer userIndexMu.RUnlock()
	_, ok := userByName[username]
	return ok
}

// NicknameExists reports whether a nickname is taken.
func NicknameExists(nickname string) bool {
	userIndexMu.RLock()
	defer userIndexMu.RUnlock()
	_, ok := userByNick[nickname]
	return ok
}

// LoadUserByUUID reads an account document by its storage key.
func LoadUserByUUID(uuid string) (*User, error) {
	path, err := userFilePath(uuid)
	if err != nil {
		return nil, err
	}

	Dblock.Lock()
	data, err := os.ReadFile(path)
	Dblock.Unlock()

	if err != nil {
		return nil, err
	}

	u := GetNewUser()
	if err = json.Unmarshal(data, &u); err != nil {
		return nil, err
	}
	if u.Uuid == "" {
		u.Uuid = uuid
	}
	return &u, nil
}

// LoadUser reads an account document by login name, resolving the name to its
// UUID through the in-memory index.
//
// A missing account is reported with an error that satisfies errors.Is(err,
// os.ErrNotExist).
func LoadUser(username string) (*User, error) {
	userIndexMu.RLock()
	uuid, ok := userByName[username]
	userIndexMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("storage: no account named %q: %w", username, os.ErrNotExist)
	}
	return LoadUserByUUID(uuid)
}

// CreateUser writes a brand new account document.
//
// The names are claimed in the index before anything is written, so two
// concurrent registrations of the same login name cannot both succeed.
func CreateUser(u *User) error {
	if u == nil {
		return errors.New("storage: nil user")
	}
	if u.UserName == "" || u.IngameName == "" {
		return errors.New("storage: username and nickname must not be empty")
	}
	if len(u.UserName) > maxNameLength || len(u.IngameName) > maxNameLength {
		return fmt.Errorf("storage: names must be at most %d characters", maxNameLength)
	}

	uuid, err := NewUUIDv4()
	if err != nil {
		return err
	}

	userIndexMu.Lock()
	if _, taken := userByName[u.UserName]; taken {
		userIndexMu.Unlock()
		return ErrUsernameTaken
	}
	if _, taken := userByNick[u.IngameName]; taken {
		userIndexMu.Unlock()
		return ErrNicknameTaken
	}
	userByName[u.UserName] = uuid
	userByNick[u.IngameName] = uuid
	userByUUID[uuid] = indexEntry{Username: u.UserName, Nickname: u.IngameName}
	userIndexMu.Unlock()

	u.Uuid = uuid
	if err = saveUserFile(u); err != nil {
		// Roll the claim back so the names do not stay reserved for an account
		// that was never written.
		userIndexMu.Lock()
		delete(userByName, u.UserName)
		delete(userByNick, u.IngameName)
		delete(userByUUID, uuid)
		userIndexMu.Unlock()
		return err
	}
	return nil
}

// SaveUser writes an existing account document and refreshes its index entries,
// retiring a nickname or login name the player has since changed.
func SaveUser(u *User) error {
	if u == nil {
		return errors.New("storage: nil user")
	}
	if u.Uuid == "" {
		return errors.New("storage: user has no uuid")
	}

	if err := saveUserFile(u); err != nil {
		return err
	}

	userIndexMu.Lock()
	defer userIndexMu.Unlock()

	if prev, ok := userByUUID[u.Uuid]; ok {
		if prev.Nickname != u.IngameName {
			if cur, exists := userByNick[prev.Nickname]; exists && cur == u.Uuid {
				delete(userByNick, prev.Nickname)
			}
		}
		if prev.Username != u.UserName {
			if cur, exists := userByName[prev.Username]; exists && cur == u.Uuid {
				delete(userByName, prev.Username)
			}
		}
	}

	userByName[u.UserName] = u.Uuid
	userByNick[u.IngameName] = u.Uuid
	userByUUID[u.Uuid] = indexEntry{Username: u.UserName, Nickname: u.IngameName}
	return nil
}

// DeleteNicknameIndex releases a nickname so that it becomes available again.
//
// It is used by the rename item: the old nickname must stop being reserved
// before the player picks up the new one.
func DeleteNicknameIndex(nickname string) error {
	userIndexMu.Lock()
	defer userIndexMu.Unlock()

	uuid, ok := userByNick[nickname]
	if !ok {
		return fmt.Errorf("storage: nickname %q is not indexed", nickname)
	}

	delete(userByNick, nickname)

	if entry, ok := userByUUID[uuid]; ok {
		entry.Nickname = ""
		userByUUID[uuid] = entry
	}
	return nil
}

// AccountRef is the cheap, index-level view of an account: enough to list every
// account without reading a single document off disk.
type AccountRef struct {
	Uuid       string `json:"uuid"`
	UserName   string `json:"username"`
	IngameName string `json:"nickname"`
}

// ListAccounts returns every known account, ordered by login name.
func ListAccounts() []AccountRef {
	userIndexMu.RLock()
	defer userIndexMu.RUnlock()

	refs := make([]AccountRef, 0, len(userByUUID))
	for uuid, entry := range userByUUID {
		refs = append(refs, AccountRef{
			Uuid:       uuid,
			UserName:   entry.Username,
			IngameName: entry.Nickname,
		})
	}

	slices.SortFunc(refs, func(a, b AccountRef) int { return cmp.Compare(a.UserName, b.UserName) })
	return refs
}

// AccountCount returns how many accounts exist.
func AccountCount() int {
	userIndexMu.RLock()
	defer userIndexMu.RUnlock()
	return len(userByUUID)
}

// DeleteUser removes an account document and its index entries.
//
// A missing account is reported with an error satisfying errors.Is(err,
// os.ErrNotExist).
func DeleteUser(username string) error {
	userIndexMu.Lock()
	uuid, ok := userByName[username]
	if !ok {
		userIndexMu.Unlock()
		return fmt.Errorf("storage: no account named %q: %w", username, os.ErrNotExist)
	}

	entry := userByUUID[uuid]
	delete(userByName, username)
	if entry.Nickname != "" {
		delete(userByNick, entry.Nickname)
	}
	delete(userByUUID, uuid)
	userIndexMu.Unlock()

	path, err := userFilePath(uuid)
	if err != nil {
		return err
	}

	Dblock.Lock()
	defer Dblock.Unlock()

	if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func saveUserFile(u *User) error {
	path, err := userFilePath(u.Uuid)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(u, "", "     ")
	if err != nil {
		return err
	}

	Dblock.Lock()
	defer Dblock.Unlock()
	return writeFileAtomic(path, data)
}

// LoadUserForLogin loads an account and verifies the password.
//
// Accounts still on the old unsalted md5 scheme are transparently rehashed the
// first time they log in successfully.
func LoadUserForLogin(loginname string, passwd []byte) (*User, int) {
	u, err := LoadUser(loginname)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, USER_NOT_FOUND
		}
		DebugInfo(1, "Cannot read account", loginname+":", err)
		return nil, USER_UNKOWN_ERROR
	}

	password := string(passwd)
	ok, needsUpgrade := VerifyPassword(string(u.Password), loginname, password)
	if !ok {
		return nil, USER_PASSWD_ERROR
	}

	u.SetID(GetNewUserID())

	if needsUpgrade {
		upgradePasswordHash(u, password)
	}

	DebugInfo(1, "User", u.UserName, "data found !")
	return u, USER_LOGIN_SUCCESS
}

// upgradePasswordHash replaces a legacy md5 hash with a salted PBKDF2 one.
func upgradePasswordHash(u *User, password string) {
	newHash, err := NewPasswordHash(password)
	if err != nil {
		DebugInfo(1, "Cannot rehash password for", u.UserName, ":", err)
		return
	}

	u.Password = []byte(newHash)

	if err = SaveUser(u); err != nil {
		DebugInfo(1, "Cannot persist rehashed password for", u.UserName, ":", err)
		return
	}

	DebugInfo(1, "Upgraded password hash for user", u.UserName)
}

// SaveAllUsers persists every account currently online.
func SaveAllUsers() bool {
	// Snapshot first: holding the manager lock across an unbounded number of
	// disk writes would stall every other goroutine.
	for _, u := range UsersManager.SnapshotUsers() {
		if err := SaveUser(u); err != nil {
			DebugInfo(1, "Cannot save user", u.UserName, ":", err)
			return false
		}
	}
	return true
}
