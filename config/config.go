package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	. "cso2-server/blademaster/typestruct"
	"cso2-server/builtin"
	. "cso2-server/kerlong"
	. "cso2-server/logging"
)

// ConfigFileName is the name looked up inside DATA_DIR, and the name of the
// copy compiled into the binary.
const ConfigFileName = "server.yaml"

// ServerVersion is reported by the startup banner and the admin status endpoint.
const ServerVersion = "v0.9.0"

// CSO2Conf is what the configuration file controls.
//
// Everything here is a gameplay or behaviour tweak. Everything that describes
// how a particular instance is deployed - ports, locale, data directory - is an
// environment variable instead, see Environment.
type CSO2Conf struct {
	MaxUsers         uint32
	EnableShop       bool
	UnlockAllWeapons bool
	DebugLevel       uint32
}

// CSO2Locales holds the strings the server sends to players, loaded from the
// locale file selected by LOCALE.
type CSO2Locales struct {
	GAME_SERVER_ERROR           string
	GAME_LOGIN_ALREADY          string
	GAME_LOGIN_EXIT_FORCE       string
	GAME_LOGIN_ERROR            string
	GAME_ROOM_COUNT_MODE_ERROR  string
	GAME_ROOM_JOIN_ERROR        string
	GAME_GM_ADD_ALLWEAPONS      string
	GAME_CHANNEL_MESSAGE_NOT_IN string
	GAME_GM_NO_AUTHORIZE        string
	GAME_USER_NEW_ITEM          string
	GAME_USER_NEW_ITEM_RESTART  string
	GAME_CHANNEL_MESSAGE        string

	MOTD []byte
}

// ConfigSource is a human-readable description of where the active
// configuration was read from, for the startup log and the admin status API.
var ConfigSource = "built-in defaults"

// fileConfig mirrors the YAML document. Keeping it separate from CSO2Conf means
// the file layout can use friendly nested names without the rest of the program
// having to know about them.
//
// The sections are named types rather than anonymous structs so that a
// misspelled key reports "not found in type config.serverSection" instead of
// dumping the whole anonymous type.
type fileConfig struct {
	Server serverSection `yaml:"server"`
	Debug  debugSection  `yaml:"debug"`
}

type serverSection struct {
	MaxUsers         uint32 `yaml:"max_users"`
	EnableShop       bool   `yaml:"enable_shop"`
	UnlockAllWeapons bool   `yaml:"unlock_all_weapons"`
}

type debugSection struct {
	Level uint32 `yaml:"level"`
}

var (
	Conf    CSO2Conf
	Locales CSO2Locales

	// Paths are relative to the built-in resource tree (see package builtin).
	localesDir = "locales/"
	motdPath   = "locales/motd.txt"
)

// InitConf loads the configuration.
//
// The copy in dataDir is overlaid on top of the copy compiled into the binary,
// so the operator's file only has to mention what it wants to change and
// everything else keeps the compiled-in default.
func (conf *CSO2Conf) InitConf(dataDir string) {
	if conf == nil {
		return
	}

	embedded, err := builtin.FS.ReadFile(ConfigFileName)
	if err != nil {
		// The embedded copy cannot be missing unless the build is broken, and
		// starting with no configuration at all would be worse than stopping.
		panic("the built-in " + ConfigFileName + " is missing from this build: " + err.Error())
	}

	file := fileConfig{}
	decodeConfig(embedded, &file, "the built-in "+ConfigFileName)

	ConfigSource = "built-in defaults"

	if dataDir != "" {
		path := filepath.Join(dataDir, ConfigFileName)
		if override, readErr := os.ReadFile(path); readErr == nil {
			ConfigSource = path
			decodeConfig(override, &file, path)
		} else if !os.IsNotExist(readErr) {
			Warn("Cannot read", path+":", readErr, "- using the built-in configuration")
		}
	}

	Info("Reading configuration from", ConfigSource)

	conf.MaxUsers = file.Server.MaxUsers
	conf.EnableShop = file.Server.EnableShop
	conf.UnlockAllWeapons = file.Server.UnlockAllWeapons
	conf.DebugLevel = file.Debug.Level
	if conf.DebugLevel > 2 {
		Warn("debug.level is", conf.DebugLevel, "- clamping to 2")
		conf.DebugLevel = 2
	}
}

// decodeConfig merges one YAML document into into.
//
// An unknown key is an error rather than a silently ignored line: that is how a
// server ends up running with settings nobody intended.
func decodeConfig(data []byte, into *fileConfig, source string) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(into); err != nil {
		// An empty file means "change nothing", which is a legitimate way to
		// say "start from the defaults".
		if errors.Is(err, io.EOF) {
			return
		}
		panic("cannot parse " + source + ": " + err.Error())
	}
}

func (locales *CSO2Locales) InitLocales() bool {
	if locales == nil {
		return false
	}

	name := Env.LocaleFileName()
	Info("Reading locale <" + name + "> ...")
	ini_parser := IniParser{}

	data, err := builtin.ReadFile(localesDir + name)
	if err == nil {
		err = ini_parser.Parse(data)
	}
	if err != nil {
		Error("Loading locale file failed:", err)
		if names := AvailableLocales(); len(names) > 0 {
			Error("LOCALE is " + strconv.Quote(Env.Locale) + "; available locales are: " + strings.Join(names, ", "))
		}
		Warn("Using default data ...")
		return false
	}

	locales.GAME_SERVER_ERROR = ini_parser.IniGetString("System", "GAME_SERVER_ERROR")
	locales.GAME_LOGIN_ALREADY = ini_parser.IniGetString("System", "GAME_LOGIN_ALREADY")
	locales.GAME_LOGIN_EXIT_FORCE = ini_parser.IniGetString("System", "GAME_LOGIN_EXIT_FORCE")
	locales.GAME_LOGIN_ERROR = ini_parser.IniGetString("System", "GAME_LOGIN_ERROR")
	locales.GAME_ROOM_COUNT_MODE_ERROR = ini_parser.IniGetString("System", "GAME_ROOM_COUNT_MODE_ERROR")
	locales.GAME_ROOM_JOIN_ERROR = ini_parser.IniGetString("System", "GAME_ROOM_JOIN_ERROR")
	locales.GAME_GM_ADD_ALLWEAPONS = ini_parser.IniGetString("System", "GAME_GM_ADD_ALLWEAPONS")
	locales.GAME_CHANNEL_MESSAGE_NOT_IN = ini_parser.IniGetString("System", "GAME_CHANNEL_MESSAGE_NOT_IN")
	locales.GAME_GM_NO_AUTHORIZE = ini_parser.IniGetString("System", "GAME_GM_NO_AUTHORIZE")
	locales.GAME_USER_NEW_ITEM = ini_parser.IniGetString("System", "GAME_USER_NEW_ITEM")
	locales.GAME_USER_NEW_ITEM_RESTART = ini_parser.IniGetString("System", "GAME_USER_NEW_ITEM_RESTART")
	locales.GAME_CHANNEL_MESSAGE = ini_parser.IniGetString("System", "GAME_CHANNEL_MESSAGE")
	return true
}

// AvailableLocales lists the locale names compiled into the binary, without
// their .ini suffix.
func AvailableLocales() []string {
	entries, err := builtin.FS.ReadDir(strings.TrimSuffix(localesDir, "/"))
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ini") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".ini"))
	}
	slices.Sort(names)
	return names
}

func (locales *CSO2Locales) InitMotd() {
	if locales == nil {
		return
	}
	Info("Reading motd ...")

	data, err := builtin.ReadFile(motdPath)
	if err != nil {
		Warn("Using default motd ...")
		locales.MOTD = []byte("You are playing Counter-Strike Online 2 on CSO2-Server.Please delete in 24 hours after loading !Visit the github web site github.com/KouKouChan/CSO2-Server")
		return
	}
	locales.MOTD = data
}

func SetLocales() {
	Info("Setting locale <" + Env.LocaleFileName() + "> ...")
	GAME_SERVER_ERROR = []byte(Locales.GAME_SERVER_ERROR)
	GAME_LOGIN_ALREADY = []byte(Locales.GAME_LOGIN_ALREADY)
	GAME_LOGIN_EXIT_FORCE = []byte(Locales.GAME_LOGIN_EXIT_FORCE)
	GAME_LOGIN_ERROR = []byte(Locales.GAME_LOGIN_ERROR)
	GAME_ROOM_COUNT_MODE_ERROR = []byte(Locales.GAME_ROOM_COUNT_MODE_ERROR)
	GAME_ROOM_JOIN_ERROR = []byte(Locales.GAME_ROOM_JOIN_ERROR)
	GAME_GM_ADD_ALLWEAPONS = []byte(Locales.GAME_GM_ADD_ALLWEAPONS)
	GAME_CHANNEL_MESSAGE_NOT_IN = []byte(Locales.GAME_CHANNEL_MESSAGE_NOT_IN)
	GAME_GM_NO_AUTHORIZE = []byte(Locales.GAME_GM_NO_AUTHORIZE)
	GAME_USER_NEW_ITEM = []byte(Locales.GAME_USER_NEW_ITEM)
	GAME_USER_NEW_ITEM_RESTART = []byte(Locales.GAME_USER_NEW_ITEM_RESTART)
	GAME_CHANNEL_MESSAGE = Locales.GAME_CHANNEL_MESSAGE
}

// Environment is the set of settings that come from environment variables.
//
// These describe how this instance is deployed, which is why they are not in
// the configuration file: the same file should produce the same server
// behaviour on any host, and a container is reconfigured by changing its
// environment rather than by rewriting a file inside the image.
type Environment struct {
	// DataDir is where player data lives and where an optional server.yaml is
	// looked up.
	DataDir string

	TCPPort   uint32
	UDPPort   uint32
	AdminPort uint32

	// EnableAdmin controls whether the admin web API is started at all.
	EnableAdmin bool

	// Locale is a bare locale name such as "zh-cn".
	Locale string

	// CodePage is the character set the game client speaks.
	CodePage string
}

var Env Environment

// Defaults for the environment variables.
const (
	DefaultTCPPort   = 30001
	DefaultUDPPort   = 30002
	DefaultAdminPort = 1315
	DefaultLocale    = "zh-cn"
)

// LocaleFileName is the locale as a file name inside the locales directory,
// e.g. "zh-cn" -> "zh-cn.ini".
func (env *Environment) LocaleFileName() string {
	name := strings.TrimSuffix(strings.TrimSpace(env.Locale), ".ini")
	if name == "" {
		name = DefaultLocale
	}
	return name + ".ini"
}

// ReadEnv loads every environment variable the server understands.
//
// A relative DATA_DIR is left as-is: it is resolved against the working
// directory later, by InitDataDir.
func (env *Environment) ReadEnv() {
	if env == nil {
		return
	}

	env.DataDir = strings.TrimSpace(os.Getenv("DATA_DIR"))
	if env.DataDir == "" {
		env.DataDir = "/data"
	}

	env.TCPPort = envPort("TCP_PORT", DefaultTCPPort)
	env.UDPPort = envPort("UDP_PORT", DefaultUDPPort)
	env.AdminPort = envPort("ADMIN_PORT", DefaultAdminPort)

	env.EnableAdmin = envBool("ENABLE_ADMIN", true)

	env.Locale = strings.TrimSpace(os.Getenv("LOCALE"))
	if env.Locale == "" {
		env.Locale = DefaultLocale
	}

	if explicit := strings.TrimSpace(os.Getenv("CODE_PAGE")); explicit != "" {
		env.CodePage = explicit
	} else {
		env.CodePage = CodePageForLocale(env.Locale)
	}
}

// envPort reads a port number, keeping the default if it is unset or nonsense.
func envPort(name string, fallback uint32) uint32 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || value == 0 || value > 65535 {
		Warn(name + " is " + strconv.Quote(raw) + ", which is not a valid port; using " + strconv.Itoa(int(fallback)))
		return fallback
	}
	return uint32(value)
}

// envBool reads a boolean. "0", "false", "no" and "off" are false; "1",
// "true", "yes" and "on" are true; anything else keeps the default.
func envBool(name string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch raw {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		Warn(name + " is " + strconv.Quote(raw) + ", which is not a boolean; using " + strconv.FormatBool(fallback))
		return fallback
	}
}

// CodePageForLocale guesses the client character set from the locale name.
//
// The client's encoding follows the game client's language, so having to state
// it twice - once as the locale, once as a code page - was a chance to get it
// wrong. CODE_PAGE still overrides this for unusual combinations.
func CodePageForLocale(locale string) string {
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(locale), ".ini"))

	switch name {
	case "zh-cn", "zh-hans", "schinese":
		return "gbk"
	case "zh-tw", "zh-hk", "zh-hant", "tchinese":
		return "big5"
	case "ko-kr", "korean":
		return "euc-kr"
	case "ja-jp", "japanese":
		return "shift_jis"
	default:
		// en-us, tr-tur, global and anything else: no conversion.
		return "utf-8"
	}
}

// DescribeEnv is a short summary of the deployment settings, for the log.
func (env *Environment) DescribeEnv() string {
	return fmt.Sprintf("tcp=%d udp=%d admin=%d enableAdmin=%t locale=%s codePage=%s",
		env.TCPPort, env.UDPPort, env.AdminPort, env.EnableAdmin, env.Locale, env.CodePage)
}
