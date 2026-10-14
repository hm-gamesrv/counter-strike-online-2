package kerlong

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempIni(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.conf")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("cannot write test config: %v", err)
	}
	return path
}

func TestLoadIniMissingFileReturnsError(t *testing.T) {
	var parser IniParser
	if err := parser.LoadIni(filepath.Join(t.TempDir(), "nope.conf")); err == nil {
		t.Fatal("loading a missing file must return an error")
	}
	// Queries against a failed load must return zero values, not panic.
	if got := parser.IniGetString("Server", "TCPPort"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if got := parser.IniGetUint32("Server", "TCPPort"); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestLoadIniParsesRealConfigShape(t *testing.T) {
	path := writeTempIni(t, "\ufeff################################\n"+
		"# comment with = inside\n"+
		"[Server]\n"+
		"TCPPort=30001\n"+
		"UDPPort=30002\n"+
		"EnableShop=1\n"+
		"\n"+
		"; another comment style\n"+
		"[Locale]\n"+
		"LocaleFile=zh-cn.ini\n"+
		"\n"+
		"[Console]\n"+
		"GMport=1315\n"+
		"GMusername=admin\n")

	var parser IniParser
	if err := parser.LoadIni(path); err != nil {
		t.Fatalf("LoadIni failed: %v", err)
	}

	if got := parser.IniGetString("Locale", "LocaleFile"); got != "zh-cn.ini" {
		t.Errorf("LocaleFile = %q, want %q", got, "zh-cn.ini")
	}
	if got := parser.IniGetUint32("Console", "GMport"); got != 1315 {
		t.Errorf("GMport = %d, want 1315", got)
	}
	if got := parser.IniGetString("Console", "GMusername"); got != "admin" {
		t.Errorf("GMusername = %q, want %q", got, "admin")
	}
	if got := parser.IniGetUint32("Server", "EnableShop"); got != 1 {
		t.Errorf("EnableShop = %d, want 1", got)
	}
	if got := parser.IniGetUint32("Server", "TCPPort"); got != 30001 {
		t.Errorf("TCPPort = %d, want 30001", got)
	}
	if got := parser.IniGetUint32("Server", "UDPPort"); got != 30002 {
		t.Errorf("UDPPort = %d, want 30002", got)
	}

	// Missing keys and sections must yield zero values, never a panic: the
	// server relies on this for optional settings.
	if got := parser.IniGetString("Server", "DoesNotExist"); got != "" {
		t.Errorf("missing key = %q, want empty", got)
	}
	if got := parser.IniGetUint32("NoSuchSection", "Key"); got != 0 {
		t.Errorf("missing section = %d, want 0", got)
	}
}

func TestLoadIniIsCaseInsensitiveAndToleratesSpacing(t *testing.T) {
	path := writeTempIni(t, "[Encode]\n   CodePage   =   gbk   \n")

	var parser IniParser
	if err := parser.LoadIni(path); err != nil {
		t.Fatalf("LoadIni failed: %v", err)
	}

	if got := parser.IniGetString("ENCODE", "codepage"); got != "gbk" {
		t.Errorf("CodePage = %q, want %q", got, "gbk")
	}
}

// A value containing '=' must keep everything after the first separator.
// Synthetic section and key: this is a parser-level property, not a real
// setting.
func TestLoadIniKeepsEqualsInValue(t *testing.T) {
	path := writeTempIni(t, "[Parser]\nValueWithEquals=abc=def=ghi\n")

	var parser IniParser
	if err := parser.LoadIni(path); err != nil {
		t.Fatalf("LoadIni failed: %v", err)
	}

	if got := parser.IniGetString("Parser", "ValueWithEquals"); got != "abc=def=ghi" {
		t.Errorf("ValueWithEquals = %q, want %q", got, "abc=def=ghi")
	}
}

// Parsing content that is already in memory is what the server does now: its
// configuration and locale files are compiled into the binary.
func TestParseInMemoryContent(t *testing.T) {
	data := []byte("\ufeff[Server]\nTCPPort=30001\n# comment\n[Admin]\nAdminPort=1315\n")

	var parser IniParser
	if err := parser.Parse(data); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if got := parser.IniGetUint32("Server", "TCPPort"); got != 30001 {
		t.Errorf("TCPPort = %d, want 30001", got)
	}
	if got := parser.IniGetUint32("Admin", "AdminPort"); got != 1315 {
		t.Errorf("AdminPort = %d, want 1315", got)
	}
}

// Parsing embedded bytes must behave exactly like parsing the same bytes from a
// file, since the real resources now take the in-memory path.
func TestParseMatchesLoadIni(t *testing.T) {
	content := "\ufeff[Server]\nTCPPort=30001\nUDPPort=30002\n\n[Locale]\nLocaleFile=zh-cn.ini\n"

	path := writeTempIni(t, content)

	var fromFile IniParser
	if err := fromFile.LoadIni(path); err != nil {
		t.Fatalf("LoadIni failed: %v", err)
	}

	var fromMemory IniParser
	if err := fromMemory.Parse([]byte(content)); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	for _, key := range [][2]string{
		{"Server", "TCPPort"},
		{"Server", "UDPPort"},
		{"Locale", "LocaleFile"},
	} {
		section, name := key[0], key[1]
		if a, b := fromFile.IniGetString(section, name), fromMemory.IniGetString(section, name); a != b {
			t.Errorf("%s/%s: file gave %q but memory gave %q", section, name, a, b)
		}
	}
}
