package kerlong

import (
	"regexp"
	"strings"
	"testing"
)

var uuidV4Pattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewUUIDv4Format(t *testing.T) {
	seen := make(map[string]bool, 1000)

	for i := 0; i < 1000; i++ {
		id, err := NewUUIDv4()
		if err != nil {
			t.Fatalf("NewUUIDv4 failed: %v", err)
		}

		if !uuidV4Pattern.MatchString(id) {
			t.Fatalf("%q is not a canonical v4 UUID", id)
		}
		if seen[id] {
			t.Fatalf("NewUUIDv4 produced a duplicate: %q", id)
		}
		seen[id] = true
	}
}

func TestIsUUID(t *testing.T) {
	valid, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4 failed: %v", err)
	}

	if !IsUUID(valid) {
		t.Errorf("IsUUID(%q) = false, want true", valid)
	}
	if !IsUUID(strings.ToUpper(valid)) {
		t.Errorf("IsUUID rejected an upper-case UUID")
	}

	invalid := []string{
		"",
		"alice",
		"../evil",
		"..\\evil",
		"../../etc/passwd",
		// Right length, but the last character is not hex. This must be built
		// from a fixed suffix rather than by trimming the random UUID, which
		// would silently degenerate into the valid UUID whenever the random one
		// happened to end in that character.
		valid[:len(valid)-1] + "z",
		valid + "-extra",
		strings.Replace(valid, "-", "", 1),    // dashes stripped
		strings.Replace(valid, "-", "_", 1),   // wrong separator
		strings.Repeat("a", 36),               // hex but wrong shape
		"00000000-0000-4000-8000-00000000000", // 35 chars
	}

	for _, s := range invalid {
		if IsUUID(s) {
			t.Errorf("IsUUID(%q) = true, want false", s)
		}
	}
}
