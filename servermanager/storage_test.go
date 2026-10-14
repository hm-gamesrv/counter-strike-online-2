package servermanager

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	. "github.com/KouKouChan/CSO2-Server/blademaster/typestruct"
	. "github.com/KouKouChan/CSO2-Server/kerlong"
	. "github.com/KouKouChan/CSO2-Server/logging"
)

// newTestStore points the storage layer at a fresh temporary DATA_DIR.
func newTestStore(t *testing.T) string {
	t.Helper()

	// Keep the JSON logger quiet while testing (DebugInfo only).
	Level = 0

	dir := t.TempDir()
	if err := InitDataDir(dir); err != nil {
		t.Fatalf("InitDataDir failed: %v", err)
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("cannot resolve temp dir: %v", err)
	}
	if DataDir != abs {
		t.Fatalf("DataDir = %q, want %q", DataDir, abs)
	}
	return abs
}

func makeUser(t *testing.T, username, nickname, password, mail string) *User {
	t.Helper()

	hash, err := NewPasswordHash(password)
	if err != nil {
		t.Fatalf("NewPasswordHash failed: %v", err)
	}

	u := GetNewUser()
	u.SetUserName(username, nickname)
	u.Password = []byte(hash)
	u.UserMail = mail
	return &u
}

// accountFiles lists the account documents currently on disk.
func accountFiles(t *testing.T, root string) []string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(root, "users"))
	if err != nil {
		t.Fatalf("cannot read users dir: %v", err)
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestInitDataDirCreatesSubdirectories(t *testing.T) {
	root := newTestStore(t)

	for _, sub := range []string{"users", "reports"} {
		info, err := os.Stat(filepath.Join(root, sub))
		if err != nil {
			t.Fatalf("%s was not created: %v", sub, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", sub)
		}
	}
}

func TestCreateAndLoadUser(t *testing.T) {
	newTestStore(t)

	created := makeUser(t, "alice", "AliceInGame", "correct-horse", "alice@example.com")
	created.Level = 7
	created.Points = 12345

	if err := CreateUser(created); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if !UserExists("alice") {
		t.Error("UserExists returned false for a just-created account")
	}
	if !NicknameExists("AliceInGame") {
		t.Error("NicknameExists returned false for a just-created nickname")
	}
	if UserExists("bob") {
		t.Error("UserExists returned true for an account that does not exist")
	}

	loaded, err := LoadUser("alice")
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}
	if loaded.UserName != "alice" || loaded.IngameName != "AliceInGame" {
		t.Errorf("round trip lost the names: %+v", loaded)
	}
	if loaded.Level != 7 || loaded.Points != 12345 {
		t.Errorf("round trip lost the progress: level=%d points=%d", loaded.Level, loaded.Points)
	}
	if loaded.UserMail != "alice@example.com" {
		t.Errorf("round trip lost the mail: %q", loaded.UserMail)
	}
	if loaded.Uuid != created.Uuid {
		t.Errorf("Uuid = %q, want %q", loaded.Uuid, created.Uuid)
	}
}

// The account document must be named after the generated UUID, and the UUID
// must survive a round trip.
func TestAccountFileIsNamedAfterUUID(t *testing.T) {
	root := newTestStore(t)

	u := makeUser(t, "alice", "AliceInGame", "pw", "a@b.c")
	if err := CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if !IsUUID(u.Uuid) {
		t.Fatalf("generated Uuid %q is not a canonical UUID", u.Uuid)
	}

	files := accountFiles(t, root)
	if len(files) != 1 {
		t.Fatalf("expected exactly 1 account file, got %v", files)
	}
	if files[0] != u.Uuid+".json" {
		t.Errorf("account file is %q, want %q", files[0], u.Uuid+".json")
	}

	// A login name must never be used as a path.
	if _, err := os.Stat(filepath.Join(root, "users", "alice")); !os.IsNotExist(err) {
		t.Error("a file named after the login name was created")
	}
}

func TestDistinctAccountsGetDistinctUUIDs(t *testing.T) {
	newTestStore(t)

	u1 := makeUser(t, "alice", "AliceInGame", "pw", "a@b.c")
	u2 := makeUser(t, "bob", "BobInGame", "pw", "b@b.c")
	if err := CreateUser(u1); err != nil {
		t.Fatalf("CreateUser(alice) failed: %v", err)
	}
	if err := CreateUser(u2); err != nil {
		t.Fatalf("CreateUser(bob) failed: %v", err)
	}

	if u1.Uuid == u2.Uuid {
		t.Fatal("two accounts were given the same uuid")
	}
}

func TestSaveUserOverwritesAtomically(t *testing.T) {
	root := newTestStore(t)

	u := makeUser(t, "alice", "AliceInGame", "pw", "a@b.c")
	if err := CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	for i := 1; i <= 3; i++ {
		u.Level = uint16(i)
		if err := SaveUser(u); err != nil {
			t.Fatalf("SaveUser #%d failed: %v", i, err)
		}
	}

	loaded, err := LoadUser("alice")
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}
	if loaded.Level != 3 {
		t.Errorf("Level = %d, want 3", loaded.Level)
	}

	for _, name := range accountFiles(t, root) {
		if filepath.Ext(name) == ".tmp" {
			t.Errorf("leftover temporary file: %s", name)
		}
	}
}

func TestStoredDocumentIsReadableJSON(t *testing.T) {
	root := newTestStore(t)

	u := makeUser(t, "alice", "AliceInGame", "pw", "a@b.c")
	if err := CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "users", u.Uuid+".json"))
	if err != nil {
		t.Fatalf("cannot read account file: %v", err)
	}

	var raw map[string]interface{}
	if err = json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("account file is not valid JSON: %v", err)
	}

	if _, present := raw["Uuid"]; !present {
		t.Error("the Uuid is missing from the document")
	}
	if _, present := raw["Inventory"]; !present {
		t.Error("the inventory is missing from the document")
	}

	// Runtime-only fields must not be persisted.
	for _, field := range []string{"Userid", "CurrentConnection", "CurrentRoomId", "NetInfo"} {
		if _, present := raw[field]; present {
			t.Errorf("runtime field %q was persisted", field)
		}
	}
}

func TestDuplicateNamesAreRejected(t *testing.T) {
	newTestStore(t)

	if err := CreateUser(makeUser(t, "alice", "AliceInGame", "pw", "a@b.c")); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	err := CreateUser(makeUser(t, "alice", "SomeoneElse", "pw", "x@y.z"))
	if !errors.Is(err, ErrUsernameTaken) {
		t.Errorf("duplicate login name: err = %v, want ErrUsernameTaken", err)
	}

	err = CreateUser(makeUser(t, "carol", "AliceInGame", "pw", "x@y.z"))
	if !errors.Is(err, ErrNicknameTaken) {
		t.Errorf("duplicate nickname: err = %v, want ErrNicknameTaken", err)
	}
}

// The names are claimed in the index before anything is written, so a race
// between two registrations of the same name cannot produce two accounts.
func TestConcurrentCreateUserOnlyOneWins(t *testing.T) {
	root := newTestStore(t)

	const workers = 16
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(n int) {
			defer wg.Done()
			u := makeUser(t, "contested", "Nick", "pw", "a@b.c")
			if err := CreateUser(u); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("%d concurrent registrations succeeded, want exactly 1", successes)
	}

	files := accountFiles(t, root)
	if len(files) != 1 {
		t.Errorf("expected 1 account file, got %v", files)
	}
}

// A failed write must not leave the names reserved.
func TestFailedCreateRollsBackTheClaim(t *testing.T) {
	root := newTestStore(t)

	u := makeUser(t, "alice", "AliceInGame", "pw", "a@b.c")

	// Make the write fail by pointing the file at a path that cannot be a
	// file: a directory named "<uuid>.json.tmp" is not creatable, so instead
	// remove the users directory.
	if err := os.RemoveAll(filepath.Join(root, "users")); err != nil {
		t.Fatalf("cannot remove users dir: %v", err)
	}

	if err := CreateUser(u); err == nil {
		t.Fatal("CreateUser succeeded even though the users directory was gone")
	}

	if UserExists("alice") {
		t.Error("the login name stayed reserved after a failed create")
	}
	if NicknameExists("AliceInGame") {
		t.Error("the nickname stayed reserved after a failed create")
	}
}

// The in-memory index must be reconstructible from the files alone.
func TestIndexIsRebuiltFromDisk(t *testing.T) {
	root := newTestStore(t)

	u := makeUser(t, "alice", "AliceInGame", "pw", "a@b.c")
	if err := CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Wipe the index, leaving the files untouched.
	userIndexMu.Lock()
	userByName = map[string]string{}
	userByNick = map[string]string{}
	userByUUID = map[string]indexEntry{}
	userIndexMu.Unlock()

	if UserExists("alice") {
		t.Fatal("index was not actually cleared")
	}

	if err := InitDataDir(root); err != nil {
		t.Fatalf("InitDataDir failed: %v", err)
	}

	if !UserExists("alice") {
		t.Error("login name was not re-indexed")
	}
	if !NicknameExists("AliceInGame") {
		t.Error("nickname was not re-indexed")
	}

	loaded, err := LoadUser("alice")
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}
	if loaded.Uuid != u.Uuid {
		t.Errorf("Uuid = %q, want %q", loaded.Uuid, u.Uuid)
	}
}

// Unreadable files must be skipped with a warning rather than breaking startup.
func TestRebuildSkipsUnreadableFiles(t *testing.T) {
	root := newTestStore(t)

	u := makeUser(t, "alice", "AliceInGame", "pw", "a@b.c")
	if err := CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	users := filepath.Join(root, "users")
	if err := os.WriteFile(filepath.Join(users, "not-json.json"), []byte("{oops"), 0644); err != nil {
		t.Fatalf("cannot write junk file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(users, "ignored.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("cannot write junk file: %v", err)
	}

	if err := InitDataDir(root); err != nil {
		t.Fatalf("InitDataDir failed: %v", err)
	}

	if !UserExists("alice") {
		t.Error("the valid account was lost while rebuilding the index")
	}
}

// Renaming a character must release the old nickname and claim the new one.
func TestRenameRetiresOldNickname(t *testing.T) {
	newTestStore(t)

	u := makeUser(t, "alice", "OldNick", "pw", "a@b.c")
	if err := CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// This mirrors what the rename item does: free the old name, then set the
	// new one and save.
	if err := DeleteNicknameIndex("OldNick"); err != nil {
		t.Fatalf("DeleteNicknameIndex failed: %v", err)
	}
	if NicknameExists("OldNick") {
		t.Error("the old nickname is still reserved")
	}

	u.SetUserName(u.UserName, "NewNick")
	if err := SaveUser(u); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	if !NicknameExists("NewNick") {
		t.Error("the new nickname was not indexed")
	}
	if NicknameExists("OldNick") {
		t.Error("the old nickname came back after saving")
	}

	loaded, err := LoadUser("alice")
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}
	if loaded.IngameName != "NewNick" {
		t.Errorf("IngameName = %q, want %q", loaded.IngameName, "NewNick")
	}
}

func TestCreateUserValidatesInput(t *testing.T) {
	newTestStore(t)

	cases := []struct {
		name     string
		username string
		nickname string
	}{
		{"empty username", "", "Nick"},
		{"empty nickname", "alice", ""},
		{"over-long username", strings.Repeat("a", maxNameLength+1), "Nick"},
		{"over-long nickname", "alice", strings.Repeat("n", maxNameLength+1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := makeUser(t, tc.username, tc.nickname, "pw", "a@b.c")
			if err := CreateUser(u); err == nil {
				t.Fatal("CreateUser accepted invalid input")
			}
			if UserExists(tc.username) || NicknameExists(tc.nickname) {
				t.Error("invalid input still claimed a name")
			}
		})
	}
}

// File names are UUIDs now, so a login name can no longer escape DATA_DIR; the
// storage key is validated instead.
func TestUUIDValidation(t *testing.T) {
	root := newTestStore(t)

	for _, bad := range []string{
		"",
		"../evil",
		"..\\evil",
		"alice",
		"not-a-uuid",
		"../../etc/passwd",
	} {
		if _, err := LoadUserByUUID(bad); !errors.Is(err, ErrInvalidUUID) {
			t.Errorf("LoadUserByUUID(%q) err = %v, want ErrInvalidUUID", bad, err)
		}
	}

	// Arbitrary strings are still safe to query: they simply miss the index.
	for _, query := range []string{"", "..", "../evil", "sub/dir", "with:colon", strings.Repeat("x", 500)} {
		if UserExists(query) {
			t.Errorf("UserExists(%q) must be false", query)
		}
		if NicknameExists(query) {
			t.Errorf("NicknameExists(%q) must be false", query)
		}
		if _, err := LoadUser(query); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("LoadUser(%q) err = %v, want os.ErrNotExist", query, err)
		}
	}

	// Nothing may have appeared outside the users directory.
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("cannot read DATA_DIR: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "users" && e.Name() != "reports" {
			t.Errorf("unexpected entry in DATA_DIR: %s", e.Name())
		}
	}
}

// Names with characters that used to be illegal in a file name are now fine,
// because names are no longer file names.
func TestNonASCIINamesAreAllowed(t *testing.T) {
	newTestStore(t)

	u := makeUser(t, "玩家一号", "狙击手", "pw", "a@b.c")
	if err := CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed for a non-ASCII name: %v", err)
	}
	if !UserExists("玩家一号") {
		t.Error("UserExists returned false for a non-ASCII account")
	}
	if !NicknameExists("狙击手") {
		t.Error("NicknameExists returned false for a non-ASCII nickname")
	}

	loaded, err := LoadUser("玩家一号")
	if err != nil {
		t.Fatalf("LoadUser failed for a non-ASCII account: %v", err)
	}
	if loaded.IngameName != "狙击手" {
		t.Errorf("IngameName = %q, want %q", loaded.IngameName, "狙击手")
	}
}

func TestLoadUserForLoginAcceptsCorrectPassword(t *testing.T) {
	newTestStore(t)

	if err := CreateUser(makeUser(t, "alice", "AliceInGame", "s3cret", "a@b.c")); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	u, status := LoadUserForLogin("alice", []byte("s3cret"))
	if status != USER_LOGIN_SUCCESS {
		t.Fatalf("status = %d, want USER_LOGIN_SUCCESS", status)
	}
	if u.UserName != "alice" {
		t.Errorf("logged in as %q, want %q", u.UserName, "alice")
	}
	if u.Userid == 0 {
		t.Error("a runtime user id should have been assigned")
	}
}

func TestLoadUserForLoginRejectsBadCredentials(t *testing.T) {
	newTestStore(t)

	if err := CreateUser(makeUser(t, "alice", "AliceInGame", "s3cret", "a@b.c")); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if _, status := LoadUserForLogin("alice", []byte("wrong")); status != USER_PASSWD_ERROR {
		t.Errorf("status = %d, want USER_PASSWD_ERROR", status)
	}
	if _, status := LoadUserForLogin("nobody", []byte("s3cret")); status != USER_NOT_FOUND {
		t.Errorf("status = %d, want USER_NOT_FOUND", status)
	}
}

// Accounts created before the PBKDF2 change stored md5(username+password) and
// must still be able to log in, with the hash upgraded on the way through.
func TestLegacyMD5LoginIsUpgradedInPlace(t *testing.T) {
	newTestStore(t)

	const username = "oldplayer"
	const nickname = "OldPlayer"
	const password = "oldpass123"

	sum := md5.Sum([]byte(username + password))
	legacy := hex.EncodeToString(sum[:])

	u := GetNewUser()
	u.SetUserName(username, nickname)
	u.Password = []byte(legacy)
	if err := CreateUser(&u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	loggedIn, status := LoadUserForLogin(username, []byte(password))
	if status != USER_LOGIN_SUCCESS {
		t.Fatalf("legacy login failed with status %d", status)
	}
	if string(loggedIn.Password) == legacy {
		t.Error("the legacy hash was not upgraded in memory")
	}

	// The upgrade must have been written to disk.
	reloaded, err := LoadUser(username)
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}
	if string(reloaded.Password) == legacy {
		t.Fatal("the legacy hash is still on disk")
	}

	ok, needsUpgrade := VerifyPassword(string(reloaded.Password), username, password)
	if !ok {
		t.Error("the stored hash no longer matches the password")
	}
	if needsUpgrade {
		t.Error("the stored hash is still flagged as legacy")
	}
}

func TestDeleteNicknameIndexRejectsUnknownNickname(t *testing.T) {
	newTestStore(t)

	if err := DeleteNicknameIndex("NeverExisted"); err == nil {
		t.Fatal("DeleteNicknameIndex accepted a nickname that was never indexed")
	}
}
