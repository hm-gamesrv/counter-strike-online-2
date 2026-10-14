package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/KouKouChan/CSO2-Server/blademaster/typestruct"
	"github.com/KouKouChan/CSO2-Server/builtin"
	. "github.com/KouKouChan/CSO2-Server/logging"
)

// The whole point of the built-in resources: the server must be able to start
// with nothing on disk next to the executable.
func TestEmbeddedResourcesArePresent(t *testing.T) {
	for _, path := range []string{
		ConfigFileName,
		"locales/zh-cn.ini",
		"locales/en-us.ini",
		"locales/global.ini",
		"locales/motd.txt",
		"scripts/item_list.csv",
		"scripts/item_unlock.csv",
		"scripts/supplyList.csv",
		"scripts/defaultItemList.csv",
		"scripts/exp_level.csv",
		"scripts/vip_info.csv",
		"scripts/shop.csv",
	} {
		if !builtin.Exists(path) {
			t.Errorf("built-in resource %q is missing from the binary", path)
		}

		data, err := builtin.ReadFile(path)
		if err != nil {
			t.Errorf("cannot read %q: %v", path, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("built-in resource %q is empty", path)
		}
	}

	if builtin.Exists("locales/definitely-not-here.ini") {
		t.Error("Exists reported a file that does not exist")
	}
	if builtin.Exists("server.conf") {
		t.Error("the retired INI configuration is still embedded")
	}
}

func TestInitConfReadsEmbeddedDefaults(t *testing.T) {
	Level = 0

	var conf CSO2Conf
	conf.InitConf(t.TempDir()) // empty dir: no override

	if conf.MaxUsers != 0 {
		t.Errorf("MaxUsers = %d, want 0 (unlimited)", conf.MaxUsers)
	}
	if !conf.EnableShop {
		t.Error("EnableShop should be true by default")
	}
	if !conf.UnlockAllWeapons {
		t.Error("UnlockAllWeapons should be true by default")
	}
	if conf.DebugLevel != 2 {
		t.Errorf("DebugLevel = %d, want 2", conf.DebugLevel)
	}
	if ConfigSource != "built-in defaults" {
		t.Errorf("ConfigSource = %q, want the built-in default", ConfigSource)
	}
}

// A server.yaml dropped into DATA_DIR wins, so the settings can be edited
// inside a mounted volume instead of being baked into the image.
func TestConfigFileInDataDirWins(t *testing.T) {
	Level = 0

	dataDir := t.TempDir()
	content := `
server:
  max_users: 32
  enable_shop: false
  unlock_all_weapons: false
debug:
  level: 1
`
	if err := os.WriteFile(filepath.Join(dataDir, ConfigFileName), []byte(content), 0644); err != nil {
		t.Fatalf("cannot write config: %v", err)
	}

	var conf CSO2Conf
	conf.InitConf(dataDir)

	if conf.MaxUsers != 32 {
		t.Errorf("MaxUsers = %d, want 32", conf.MaxUsers)
	}
	if conf.EnableShop {
		t.Error("EnableShop = true, want false from the file")
	}
	if conf.UnlockAllWeapons {
		t.Error("UnlockAllWeapons = true, want false from the file")
	}
	if conf.DebugLevel != 1 {
		t.Errorf("DebugLevel = %d, want 1", conf.DebugLevel)
	}
	if !strings.HasSuffix(ConfigSource, ConfigFileName) {
		t.Errorf("ConfigSource = %q, want the path to the file", ConfigSource)
	}
}

// Partial files keep the compiled-in defaults for whatever they omit.
func TestPartialConfigFileKeepsDefaults(t *testing.T) {
	Level = 0

	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, ConfigFileName),
		[]byte("server:\n  enable_shop: false\n"), 0644); err != nil {
		t.Fatalf("cannot write config: %v", err)
	}

	var conf CSO2Conf
	conf.InitConf(dataDir)

	if conf.EnableShop {
		t.Error("EnableShop = true, want false from the file")
	}
	if !conf.UnlockAllWeapons {
		t.Error("UnlockAllWeapons = false, want the built-in default true")
	}
	if conf.DebugLevel != 2 {
		t.Errorf("DebugLevel = %d, want the built-in default 2", conf.DebugLevel)
	}
}

// A misspelled key must be loud. Silently ignoring it is how a server ends up
// running with settings nobody intended.
func TestUnknownConfigKeyPanics(t *testing.T) {
	Level = 0

	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, ConfigFileName),
		[]byte("server:\n  enable_shopp: true\n"), 0644); err != nil {
		t.Fatalf("cannot write config: %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Error("a misspelled key did not stop the server")
		}
	}()

	var conf CSO2Conf
	conf.InitConf(dataDir)
}

func TestInitLocalesReadsEmbeddedLocale(t *testing.T) {
	Level = 0
	builtin.Init(t.TempDir())

	Env.Locale = "zh-cn"

	var locales CSO2Locales
	if !locales.InitLocales() {
		t.Fatal("InitLocales failed for the embedded zh-cn.ini")
	}

	if locales.GAME_SERVER_ERROR == "" {
		t.Error("GAME_SERVER_ERROR is empty")
	}
	if locales.GAME_USER_NEW_ITEM == "" {
		t.Error("GAME_USER_NEW_ITEM is empty")
	}
}

// Every locale shipped in the binary must parse and carry the strings the
// server actually uses, otherwise picking it breaks at runtime.
func TestEveryEmbeddedLocaleParses(t *testing.T) {
	Level = 0
	builtin.Init(t.TempDir())

	names := AvailableLocales()
	if len(names) == 0 {
		t.Fatal("AvailableLocales returned nothing")
	}

	for _, name := range names {
		Env.Locale = name

		var locales CSO2Locales
		if !locales.InitLocales() {
			t.Errorf("locale %s failed to load", name)
			continue
		}
		if locales.GAME_SERVER_ERROR == "" {
			t.Errorf("locale %s has no GAME_SERVER_ERROR", name)
		}
	}

	t.Logf("checked %d embedded locales: %s", len(names), strings.Join(names, ", "))
}

func TestInitMotdReadsEmbeddedMotd(t *testing.T) {
	Level = 0
	builtin.Init(t.TempDir())

	var locales CSO2Locales
	locales.InitMotd()

	if len(locales.MOTD) == 0 {
		t.Error("MOTD is empty")
	}
}

func TestInitCSVReadsEmbeddedGameData(t *testing.T) {
	Level = 0
	builtin.Init(t.TempDir())

	// Reset the package-level tables so the assertions below mean something.
	ItemList = map[uint32]ItemData{}
	UnlockList = map[uint32]UnlockData{}
	BoxList = map[uint32]BoxData{}
	BoxIDs = nil
	DefaultInventoryItem = nil
	ShopItemList = nil

	Conf.EnableShop = true
	InitCSV()

	if len(ItemList) == 0 {
		t.Error("the weapon list did not load")
	}
	if len(UnlockList) == 0 {
		t.Error("the unlock list did not load")
	}
	if len(BoxList) == 0 || len(BoxIDs) == 0 {
		t.Error("the supply box list did not load")
	}
	if len(DefaultInventoryItem) == 0 {
		t.Error("the default inventory list did not load")
	}
	if len(ShopItemList) == 0 {
		t.Error("the shop list did not load")
	}

	t.Logf("items=%d unlocks=%d boxes=%d defaultItems=%d shopEntries=%d",
		len(ItemList), len(UnlockList), len(BoxList), len(DefaultInventoryItem), len(ShopItemList))
}

// The locale files are still read through the builtin override mechanism, so an
// operator can retranslate one without rebuilding.
func TestLocaleOverrideWinsOverEmbedded(t *testing.T) {
	Level = 0

	root := t.TempDir()
	overrideDir := filepath.Join(root, builtin.OverrideDir, "locales")
	if err := os.MkdirAll(overrideDir, 0755); err != nil {
		t.Fatalf("cannot create override dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(overrideDir, "zh-cn.ini"),
		[]byte("[System]\nGAME_SERVER_ERROR=OVERridden\n"), 0644); err != nil {
		t.Fatalf("cannot write override: %v", err)
	}

	builtin.Init(root)
	t.Cleanup(func() { builtin.Init(t.TempDir()) })

	Env.Locale = "zh-cn"
	var locales CSO2Locales
	if !locales.InitLocales() {
		t.Fatal("InitLocales failed with a locale override")
	}
	if locales.GAME_SERVER_ERROR != "OVERridden" {
		t.Errorf("GAME_SERVER_ERROR = %q, want the override", locales.GAME_SERVER_ERROR)
	}

	// A locale that was not overridden still loads from the binary.
	Env.Locale = "en-us"
	var other CSO2Locales
	if !other.InitLocales() {
		t.Fatal("an un-overridden locale failed to load from the embedded copy")
	}
	if other.GAME_SERVER_ERROR == "" {
		t.Error("the embedded locale came back empty")
	}
}

// --- environment ---

var allEnvNames = []string{
	"DATA_DIR", "TCP_PORT", "UDP_PORT", "ADMIN_PORT",
	"ENABLE_ADMIN", "LOCALE", "CODE_PAGE",
}

// readEnvWith starts from a clean environment, so the test does not depend on
// whatever the developer happens to have exported.
func readEnvWith(t *testing.T, vars map[string]string) Environment {
	t.Helper()

	for _, name := range allEnvNames {
		os.Unsetenv(name)
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
	for name, value := range vars {
		t.Setenv(name, value)
	}

	var env Environment
	env.ReadEnv("/exe")
	return env
}

func TestReadEnvDefaults(t *testing.T) {
	Level = 0

	env := readEnvWith(t, nil)

	if env.TCPPort != DefaultTCPPort {
		t.Errorf("TCPPort = %d, want %d", env.TCPPort, DefaultTCPPort)
	}
	if env.UDPPort != DefaultUDPPort {
		t.Errorf("UDPPort = %d, want %d", env.UDPPort, DefaultUDPPort)
	}
	if env.AdminPort != DefaultAdminPort {
		t.Errorf("AdminPort = %d, want %d", env.AdminPort, DefaultAdminPort)
	}
	if !env.EnableAdmin {
		t.Error("EnableAdmin = false, want true by default")
	}
	if env.Locale != DefaultLocale {
		t.Errorf("Locale = %q, want %q", env.Locale, DefaultLocale)
	}
	if env.CodePage != "gbk" {
		t.Errorf("CodePage = %q, want gbk derived from zh-cn", env.CodePage)
	}
	if filepath.Base(env.DataDir) != "data" {
		t.Errorf("DataDir = %q, want the default <exe>/data", env.DataDir)
	}
}

func TestReadEnvOverrides(t *testing.T) {
	Level = 0

	env := readEnvWith(t, map[string]string{
		"DATA_DIR":     filepath.Join("srv", "cso2"),
		"TCP_PORT":     "31001",
		"UDP_PORT":     "31002",
		"ADMIN_PORT":   "1316",
		"ENABLE_ADMIN": "false",
		"LOCALE":       "en-us",
	})

	if env.DataDir != filepath.Join("srv", "cso2") {
		t.Errorf("DataDir = %q", env.DataDir)
	}
	if env.TCPPort != 31001 || env.UDPPort != 31002 || env.AdminPort != 1316 {
		t.Errorf("ports = %d/%d/%d, want 31001/31002/1316", env.TCPPort, env.UDPPort, env.AdminPort)
	}
	if env.EnableAdmin {
		t.Error("EnableAdmin = true, want false")
	}
	if env.Locale != "en-us" {
		t.Errorf("Locale = %q", env.Locale)
	}
	if env.CodePage != "utf-8" {
		t.Errorf("CodePage = %q, want utf-8 derived from en-us", env.CodePage)
	}
	if env.LocaleFileName() != "en-us.ini" {
		t.Errorf("LocaleFileName = %q, want en-us.ini", env.LocaleFileName())
	}
}

// A bad port must not silently become 0 or some other nonsense value.
func TestReadEnvRejectsBadPorts(t *testing.T) {
	Level = 0

	for _, bad := range []string{"not-a-number", "0", "70000", "-1"} {
		env := readEnvWith(t, map[string]string{"TCP_PORT": bad})
		if env.TCPPort != DefaultTCPPort {
			t.Errorf("TCP_PORT=%q gave %d, want the default %d", bad, env.TCPPort, DefaultTCPPort)
		}
	}
}

func TestReadEnvRejectsBadBooleans(t *testing.T) {
	Level = 0

	env := readEnvWith(t, map[string]string{"ENABLE_ADMIN": "maybe"})
	if !env.EnableAdmin {
		t.Error("a nonsense ENABLE_ADMIN should keep the default (true)")
	}

	for _, yes := range []string{"1", "true", "TRUE", "yes", "on"} {
		env := readEnvWith(t, map[string]string{"ENABLE_ADMIN": yes})
		if !env.EnableAdmin {
			t.Errorf("ENABLE_ADMIN=%q should be true", yes)
		}
	}
	for _, no := range []string{"0", "false", "False", "no", "off"} {
		env := readEnvWith(t, map[string]string{"ENABLE_ADMIN": no})
		if env.EnableAdmin {
			t.Errorf("ENABLE_ADMIN=%q should be false", no)
		}
	}
}

// The code page follows the client's language; having to state it separately
// was a chance to get it wrong.
func TestCodePageForLocale(t *testing.T) {
	cases := map[string]string{
		"zh-cn":     "gbk",
		"zh-cn.ini": "gbk",
		"ZH-CN":     "gbk",
		"zh-tw":     "big5",
		"zh-hk":     "big5",
		"ko-kr":     "euc-kr",
		"ja-jp":     "shift_jis",
		"en-us":     "utf-8",
		"tr-tur":    "utf-8",
		"global":    "utf-8",
		"":          "utf-8",
	}

	for locale, want := range cases {
		if got := CodePageForLocale(locale); got != want {
			t.Errorf("CodePageForLocale(%q) = %q, want %q", locale, got, want)
		}
	}
}

// CODE_PAGE still overrides the derivation, for unusual combinations.
func TestCodePageEnvOverridesDerivation(t *testing.T) {
	Level = 0

	env := readEnvWith(t, map[string]string{"LOCALE": "zh-cn", "CODE_PAGE": "big5"})
	if env.CodePage != "big5" {
		t.Errorf("CodePage = %q, want the explicit override big5", env.CodePage)
	}
}

func TestLocaleFileNameHandlesExtensionsAndBlanks(t *testing.T) {
	cases := map[string]string{
		"zh-cn":     "zh-cn.ini",
		"zh-cn.ini": "zh-cn.ini",
		"  en-us  ": "en-us.ini",
		"":          DefaultLocale + ".ini",
	}

	for input, want := range cases {
		env := Environment{Locale: input}
		if got := env.LocaleFileName(); got != want {
			t.Errorf("LocaleFileName(%q) = %q, want %q", input, got, want)
		}
	}
}
