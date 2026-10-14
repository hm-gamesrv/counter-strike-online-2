package typestruct

import (
	"testing"

	. "cso2-server/blademaster/Exp"
)

// TestGetNewUserDefaults pins down the defaults produced by GetNewUser.
//
// That function builds a User with an *unkeyed* composite literal, so the
// values line up with the struct's field order. Inserting a field anywhere in
// User silently shifts every default after it — which is exactly what happened
// when the storage key (Uuid) was introduced. This test turns that class of
// mistake into a loud failure instead of a subtle gameplay bug.
func TestGetNewUserDefaults(t *testing.T) {
	u := GetNewUser()

	if u.Uuid != "" {
		t.Errorf("Uuid = %q, want empty until the account is created", u.Uuid)
	}
	if u.Userid != 0 {
		t.Errorf("Userid = %d, want 0 (assigned at login)", u.Userid)
	}
	if u.NexonID != 1 {
		t.Errorf("NexonID = %d, want 1", u.NexonID)
	}
	if u.UserName != "" || u.IngameName != "" {
		t.Errorf("names should start empty, got %q / %q", u.UserName, u.IngameName)
	}
	if len(u.Password) != 0 {
		t.Errorf("Password should start empty, got %d bytes", len(u.Password))
	}
	if u.Gm != 0 {
		t.Errorf("Gm = %d, want 0", u.Gm)
	}

	// Progression
	if u.Level != 1 {
		t.Errorf("Level = %d, want 1", u.Level)
	}
	if u.Points != 10000 {
		t.Errorf("Points = %d, want 10000", u.Points)
	}
	if u.CurrentExp != 0 {
		t.Errorf("CurrentExp = %d, want 0", u.CurrentExp)
	}
	if u.MaxExp != LevelExp[0] {
		t.Errorf("MaxExp = %d, want %d", u.MaxExp, LevelExp[0])
	}

	// Cosmetic defaults
	if u.Avatar != 1006 {
		t.Errorf("Avatar = %d, want 1006", u.Avatar)
	}

	// Skill defaults
	if u.SkillHumanMaxXp != 0x19AC {
		t.Errorf("SkillHumanMaxXp = %#x, want 0x19AC", u.SkillHumanMaxXp)
	}
	if u.SkillZombieMaxXp != 0x16F6 {
		t.Errorf("SkillZombieMaxXp = %#x, want 0x16F6", u.SkillZombieMaxXp)
	}

	// Unlock bitmaps, which is where a field shift would be hardest to notice.
	// Note: all three are 128-byte (1024-bit) bitmaps in this implementation,
	// even though the packet builders treat avatars and achievements as if
	// they could hold more entries.
	if got := len(u.UnlockedTitles); got != 128 {
		t.Errorf("len(UnlockedTitles) = %d, want 128", got)
	}
	if len(u.UnlockedAvatars) != 128 {
		t.Errorf("len(UnlockedAvatars) = %d, want 128", len(u.UnlockedAvatars))
	}
	if len(u.UnlockedAchievements) != 128 {
		t.Errorf("len(UnlockedAchievements) = %d, want 128", len(u.UnlockedAchievements))
	}
	if got := len(u.Signature); got != 4 {
		t.Errorf("len(Signature) = %d, want 4", got)
	}
	if got := len(u.NetCafeName); got != 4 {
		t.Errorf("len(NetCafeName) = %d, want 4", got)
	}
	if got := len(u.ClanName); got != 4 {
		t.Errorf("len(ClanName) = %d, want 4", got)
	}

	// The first unlocked-title byte is 0x02 in the shipped default.
	if u.UnlockedTitles[5] != 0x02 {
		t.Errorf("UnlockedTitles[5] = %#x, want 0x02", u.UnlockedTitles[5])
	}

	// Runtime state
	if u.CurrentChannelServerIndex != 1 {
		t.Errorf("CurrentChannelServerIndex = %d, want 1", u.CurrentChannelServerIndex)
	}
	if u.CurrentChannelIndex != 0 || u.CurrentRoomId != 0 {
		t.Errorf("user should start outside any channel/room, got %d/%d",
			u.CurrentChannelIndex, u.CurrentRoomId)
	}
	if u.CurrentIsIngame {
		t.Error("CurrentIsIngame should start false")
	}
	if u.CurrentConnection != nil {
		t.Error("CurrentConnection should start nil")
	}
	if u.CurrentSequence != nil {
		t.Error("CurrentSequence should start nil")
	}
	if u.UserMutex == nil {
		t.Fatal("UserMutex must be initialised: every accessor dereferences it")
	}
	if u.WeaponKills == nil {
		t.Error("WeaponKills must be initialised")
	}
	if len(u.Inventory.Items) != int(u.Inventory.NumOfItem) {
		t.Errorf("inventory item count mismatch: %d items but NumOfItem=%d",
			len(u.Inventory.Items), u.Inventory.NumOfItem)
	}
}

// Each call must produce independent state: the default bitmaps must not be
// shared between users.
func TestGetNewUserDoesNotShareState(t *testing.T) {
	a := GetNewUser()
	b := GetNewUser()

	a.UnlockedTitles[0] = 0xFF
	if b.UnlockedTitles[0] == 0xFF {
		t.Error("two users share the same UnlockedTitles backing array")
	}

	a.Inventory.Items = append(a.Inventory.Items, UserInventoryItem{Id: 1, Count: 1})
	if len(b.Inventory.Items) == len(a.Inventory.Items) {
		t.Error("two users share the same inventory slice")
	}

	if a.UserMutex == b.UserMutex {
		t.Error("two users share the same mutex")
	}
}
