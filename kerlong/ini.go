package kerlong

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"
)

// IniParser is a small, dependency-free INI reader.
//
// It intentionally supports only the subset of INI that this project's config
// and locale files actually use:
//
//	# comment            ; comment
//	[Section]
//	Key=Value            Key = Value
//
// Section and key lookups are case-insensitive. Missing sections or keys yield
// the zero value, which is the behaviour the rest of the server relies on for
// optional settings. This replaces the previous gopkg.in/ini.v1 dependency so
// that the server builds with no third-party INI library.
type IniParser struct {
	sections map[string]map[string]string
}

// IniParserError is returned when a config file cannot be read.
type IniParserError struct {
	error_info string
}

func (e *IniParserError) Error() string { return e.error_info }

// LoadIni reads and parses the given INI file.
func (this *IniParser) LoadIni(config_file_name string) error {
	data, err := os.ReadFile(config_file_name)
	if err != nil {
		this.sections = nil
		return &IniParserError{"cannot open config file: " + err.Error()}
	}

	return this.Parse(data)
}

// Parse reads INI content that is already in memory.
//
// This is what the server uses: its configuration and locale files are compiled
// into the binary, not read from disk.
func (this *IniParser) Parse(data []byte) error {
	sections := make(map[string]map[string]string)
	current := ""

	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Some locale files contain long MOTD-ish strings; raise the line limit.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	first := true
	for scanner.Scan() {
		line := scanner.Text()

		// Strip a UTF-8 BOM if the file starts with one.
		if first {
			line = strings.TrimPrefix(line, "\ufeff")
			first = false
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			if end := strings.Index(line, "]"); end > 0 {
				current = normalizeIniKey(line[1:end])
			}
			continue
		}

		sep := strings.Index(line, "=")
		if sep < 0 {
			continue
		}

		key := normalizeIniKey(line[:sep])
		value := strings.TrimSpace(line[sep+1:])
		if key == "" {
			continue
		}

		if sections[current] == nil {
			sections[current] = make(map[string]string)
		}
		sections[current][key] = value
	}

	if err := scanner.Err(); err != nil {
		this.sections = nil
		return &IniParserError{"cannot read config data: " + err.Error()}
	}

	this.sections = sections
	return nil
}

func normalizeIniKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (this *IniParser) rawValue(section string, key string) (string, bool) {
	if this.sections == nil {
		return "", false
	}
	s, ok := this.sections[normalizeIniKey(section)]
	if !ok {
		return "", false
	}
	v, ok := s[normalizeIniKey(key)]
	return v, ok
}

// IniGetString returns the key's value, or "" when absent.
func (this *IniParser) IniGetString(section string, key string) string {
	v, _ := this.rawValue(section, key)
	return v
}

// IniGetInt32 returns the key's value as an int32, or 0 when absent/invalid.
func (this *IniParser) IniGetInt32(section string, key string) int32 {
	v, _ := this.rawValue(section, key)
	n, err := strconv.ParseInt(strings.TrimSpace(v), 0, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

// IniGetUint32 returns the key's value as a uint32, or 0 when absent/invalid.
func (this *IniParser) IniGetUint32(section string, key string) uint32 {
	v, _ := this.rawValue(section, key)
	n, err := strconv.ParseUint(strings.TrimSpace(v), 0, 32)
	if err != nil {
		return 0
	}
	return uint32(n)
}

// IniGetInt64 returns the key's value as an int64, or 0 when absent/invalid.
func (this *IniParser) IniGetInt64(section string, key string) int64 {
	v, _ := this.rawValue(section, key)
	n, err := strconv.ParseInt(strings.TrimSpace(v), 0, 64)
	if err != nil {
		return 0
	}
	return n
}

// IniGetUint64 returns the key's value as a uint64, or 0 when absent/invalid.
func (this *IniParser) IniGetUint64(section string, key string) uint64 {
	v, _ := this.rawValue(section, key)
	n, err := strconv.ParseUint(strings.TrimSpace(v), 0, 64)
	if err != nil {
		return 0
	}
	return n
}

// IniGetFloat32 returns the key's value as a float32, or 0 when absent/invalid.
func (this *IniParser) IniGetFloat32(section string, key string) float32 {
	v, _ := this.rawValue(section, key)
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 32)
	if err != nil {
		return 0
	}
	return float32(f)
}

// IniGetFloat64 returns the key's value as a float64, or 0 when absent/invalid.
func (this *IniParser) IniGetFloat64(section string, key string) float64 {
	v, _ := this.rawValue(section, key)
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0
	}
	return f
}
