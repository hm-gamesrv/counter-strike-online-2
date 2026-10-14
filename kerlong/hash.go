package kerlong

import (
	"crypto/md5"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// Password hashing.
//
// The original implementation stored `md5(username + password)` — unsalted,
// fast to brute-force, and identical for two users who pick the same password.
// New and upgraded accounts now use PBKDF2-HMAC-SHA512 with a per-user random
// salt.
//
// Stored formats:
//
//	legacy:   32 hex characters, i.e. md5(username + password)
//	current:  "2:<salt hex>:<iterations>:<derived key hex>"
//
// Legacy hashes are still accepted so that existing accounts keep working, and
// VerifyPassword reports needsUpgrade so the caller can transparently rehash
// them on a successful login.
const (
	// PasswordHashVersion identifies the current storage format.
	PasswordHashVersion = 2

	passwordHashIterations = 100000
	passwordHashKeyLength  = 64
	passwordHashSaltBytes  = 16

	legacyMD5HexLength = 32
)

// PasswordHashError describes a stored hash that could not be interpreted.
type PasswordHashError struct {
	reason string
}

func (e *PasswordHashError) Error() string { return "password hash: " + e.reason }

// NewPasswordHash derives a new salted hash for the given password.
func NewPasswordHash(password string) (string, error) {
	salt := make([]byte, passwordHashSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", &PasswordHashError{"cannot generate salt: " + err.Error()}
	}

	key, err := pbkdf2.Key(sha512.New, password, salt, passwordHashIterations, passwordHashKeyLength)
	if err != nil {
		return "", &PasswordHashError{"cannot derive key: " + err.Error()}
	}

	return fmt.Sprintf("%d:%s:%d:%s",
		PasswordHashVersion,
		hex.EncodeToString(salt),
		passwordHashIterations,
		hex.EncodeToString(key),
	), nil
}

// VerifyPassword checks a password against a stored hash.
//
// username is required because the legacy format mixed it into the digest.
// needsUpgrade is true when the stored hash uses the legacy MD5 format and
// should be replaced with a call to NewPasswordHash.
func VerifyPassword(stored string, username string, password string) (ok bool, needsUpgrade bool) {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return false, false
	}

	if isLegacyPasswordHash(stored) {
		return verifyLegacyMD5(stored, username, password), true
	}

	ok = verifyPBKDF2(stored, password)
	return ok, false
}

// isLegacyPasswordHash reports whether the stored value uses the old format.
func isLegacyPasswordHash(stored string) bool {
	if len(stored) != legacyMD5HexLength {
		return false
	}
	_, err := hex.DecodeString(stored)
	return err == nil
}

// verifyLegacyMD5 reproduces the original md5(username + password) check.
func verifyLegacyMD5(stored string, username string, password string) bool {
	sum := md5.Sum([]byte(username + password))
	expected := hex.EncodeToString(sum[:])

	// Constant-time compare so that a timing side channel cannot be used to
	// recover the digest byte by byte.
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(stored))) == 1
}

// verifyPBKDF2 checks a password against the current "version:salt:iter:key"
// format.
func verifyPBKDF2(stored string, password string) bool {
	parts := strings.Split(stored, ":")
	if len(parts) != 4 {
		return false
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil || version != PasswordHashVersion {
		return false
	}

	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}

	iterations, err := strconv.Atoi(parts[2])
	if err != nil || iterations <= 0 {
		return false
	}

	expected, err := hex.DecodeString(parts[3])
	if err != nil || len(expected) == 0 {
		return false
	}

	key, err := pbkdf2.Key(sha512.New, password, salt, iterations, len(expected))
	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare(key, expected) == 1
}
