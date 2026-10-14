package kerlong

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"
)

// legacyHash reproduces exactly what the old code stored, so the compatibility
// path can be tested against real historical data.
func legacyHash(username, password string) string {
	sum := md5.Sum([]byte(username + password))
	return hex.EncodeToString(sum[:])
}

func TestNewPasswordHashRoundTrip(t *testing.T) {
	const username = "testuser"
	const password = "s3cret-p4ss"

	stored, err := NewPasswordHash(password)
	if err != nil {
		t.Fatalf("NewPasswordHash returned an error: %v", err)
	}

	if parts := strings.Split(stored, ":"); len(parts) != 4 {
		t.Fatalf("expected a 4-component hash, got %q", stored)
	}

	ok, needsUpgrade := VerifyPassword(stored, username, password)
	if !ok {
		t.Fatal("the correct password was rejected")
	}
	if needsUpgrade {
		t.Fatal("a freshly created hash must not be flagged for upgrade")
	}
}

func TestNewPasswordHashUsesRandomSalt(t *testing.T) {
	const password = "same-password"

	first, err := NewPasswordHash(password)
	if err != nil {
		t.Fatalf("NewPasswordHash returned an error: %v", err)
	}
	second, err := NewPasswordHash(password)
	if err != nil {
		t.Fatalf("NewPasswordHash returned an error: %v", err)
	}

	if first == second {
		t.Fatal("two hashes of the same password are identical: the salt is not random")
	}

	// Both must still verify.
	for _, stored := range []string{first, second} {
		if ok, _ := VerifyPassword(stored, "anyone", password); !ok {
			t.Fatal("a salted hash failed to verify")
		}
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	stored, err := NewPasswordHash("correct-horse")
	if err != nil {
		t.Fatalf("NewPasswordHash returned an error: %v", err)
	}

	if ok, _ := VerifyPassword(stored, "testuser", "battery-staple"); ok {
		t.Fatal("an incorrect password was accepted")
	}
}

// Accounts created before this change are still on md5(username+password) and
// must keep working, while being flagged for a transparent upgrade.
func TestVerifyPasswordAcceptsLegacyMD5AndFlagsUpgrade(t *testing.T) {
	const username = "oldplayer"
	const password = "oldpass123"

	stored := legacyHash(username, password)

	ok, needsUpgrade := VerifyPassword(stored, username, password)
	if !ok {
		t.Fatal("a valid legacy md5 password was rejected")
	}
	if !needsUpgrade {
		t.Fatal("a legacy md5 hash must be flagged for upgrade")
	}
}

func TestVerifyPasswordRejectsLegacyMD5WithWrongPassword(t *testing.T) {
	stored := legacyHash("oldplayer", "oldpass123")

	if ok, _ := VerifyPassword(stored, "oldplayer", "wrongpass"); ok {
		t.Fatal("a legacy hash accepted the wrong password")
	}
	// The legacy scheme mixes in the user name, so a different account name
	// must not validate either.
	if ok, _ := VerifyPassword(stored, "otherplayer", "oldpass123"); ok {
		t.Fatal("a legacy hash accepted a different user name")
	}
}

func TestVerifyPasswordHandlesMalformedHashes(t *testing.T) {
	cases := []struct {
		name   string
		stored string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"not a hash", "hunter2"},
		{"wrong component count", "2:abcd:1000"},
		{"unknown version", "99:abcd:100000:abcd"},
		{"non-numeric iterations", "2:abcd:abc:abcd"},
		{"non-hex salt", "2:zzzz:100000:abcd"},
		{"non-hex key", "2:abcd:100000:zzzz"},
		{"zero iterations", "2:abcd:0:abcd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if ok, _ := VerifyPassword(tc.stored, "user", "password"); ok {
				t.Fatalf("malformed hash %q was accepted", tc.stored)
			}
		})
	}
}
