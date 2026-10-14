package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	. "cso2-server/blademaster/typestruct"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
	. "cso2-server/servermanager"
)

// A single shutdown signal must flush every online account and let the process
// exit: there is no "press CTRL+C again" step any more.
func TestAwaitShutdownSavesOnTheFirstSignal(t *testing.T) {
	Level = 0

	dir := t.TempDir()
	if err := InitDataDir(dir); err != nil {
		t.Fatalf("InitDataDir failed: %v", err)
	}

	// Register an account and make it look online.
	hash, err := NewPasswordHash("pw")
	if err != nil {
		t.Fatalf("NewPasswordHash failed: %v", err)
	}

	u := GetNewUser()
	u.SetUserName("alice", "AliceInGame")
	u.Password = []byte(hash)
	if err = CreateUser(&u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	u.Level = 99
	if err = SaveUser(&u); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	// Put it in the online manager, in memory only, with a level the file does
	// not have yet: if the shutdown flushes, that is what lands on disk.
	// AddUser requires a runtime slot id, which is normally handed out at login.
	u.SetID(GetNewUserID())
	u.Level = 7
	if !UsersManager.AddUser(&u) {
		t.Fatal("could not add the user to the online manager")
	}
	t.Cleanup(func() { UsersManager.DelUserByID(u.Userid) })

	// Sanity: the file still holds the pre-shutdown value.
	before, err := LoadUser("alice")
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}
	if before.Level != 99 {
		t.Fatalf("setup: on-disk level = %d, want 99", before.Level)
	}

	// One signal.
	ch := make(chan os.Signal, 1)
	done := make(chan struct{})
	go func() {
		awaitShutdown(ch)
		close(done)
	}()

	ch <- syscall.SIGINT

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("awaitShutdown did not return after a single signal")
	}

	after, err := LoadUser("alice")
	if err != nil {
		t.Fatalf("LoadUser after shutdown failed: %v", err)
	}
	if after.Level != 7 {
		t.Errorf("on-disk level after shutdown = %d, want 7 (the account was not flushed)", after.Level)
	}

	// Nothing should have been left behind half-written.
	entries, err := os.ReadDir(filepath.Join(dir, "users"))
	if err != nil {
		t.Fatalf("cannot read users dir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("leftover temporary file after shutdown: %s", e.Name())
		}
	}
}

// With nobody online the shutdown must still return promptly.
func TestAwaitShutdownWithNoOnlineUsers(t *testing.T) {
	Level = 0

	if err := InitDataDir(t.TempDir()); err != nil {
		t.Fatalf("InitDataDir failed: %v", err)
	}

	ch := make(chan os.Signal, 1)
	done := make(chan struct{})
	go func() {
		awaitShutdown(ch)
		close(done)
	}()

	ch <- syscall.SIGTERM

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("awaitShutdown did not return with no online users")
	}
}
