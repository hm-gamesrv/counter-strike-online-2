// Package builtin holds the server's built-in resources and makes them
// available to the rest of the program.
//
// Everything the server needs to start — the default configuration, the locale
// files and the game data extracted from the client — lives in this directory
// and is compiled into the binary with go:embed. A release therefore consists
// of a single executable: there is no resource folder to copy alongside it.
//
// # Overriding a built-in file
//
// Operators and modders can still replace individual files without rebuilding.
// Any file found under
//
//	<executable directory>/builtin/<same relative path>
//
// takes precedence over the embedded copy. For example, dropping a
//
//	builtin/locales/ja-jp.ini
//
// next to the executable adds a language, and
//
//	builtin/locales/zh-cn.ini
//
// replaces just that one locale. Files that are not overridden keep coming from
// the binary.
//
// Configuration is deliberately not covered by this rule: it is read from
// $DATA_DIR/server.yaml instead, so that it travels with the player data.
package builtin

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"cso2-server/logging"
)

// FS is the embedded resource tree: server.yaml, locales/ and scripts/.
//
//go:embed server.yaml locales scripts
var FS embed.FS

// OverrideDir is the directory the on-disk overrides are read from, relative to
// the executable.
const OverrideDir = "builtin"

var (
	overrideRootMu sync.RWMutex
	overrideRoot   string
)

// Init records where the executable lives, so that on-disk overrides can be
// looked up next to it.
//
// It is optional: if it is never called the override root is derived from
// os.Executable on first use. Calling it again replaces the root, which is what
// the tests use.
func Init(exeDir string) {
	overrideRootMu.Lock()
	overrideRoot = exeDir
	overrideRootMu.Unlock()
}

func root() string {
	overrideRootMu.RLock()
	dir := overrideRoot
	overrideRootMu.RUnlock()

	if dir != "" {
		return dir
	}

	exe, err := os.Executable()
	if err != nil {
		return ""
	}

	dir = filepath.Dir(exe)
	Init(dir)
	return dir
}

// ReadFile returns a built-in resource by its slash-separated relative path,
// such as "server.yaml", "locales/zh-cn.ini" or "scripts/item_list.csv".
//
// An on-disk override wins over the embedded copy.
func ReadFile(relPath string) ([]byte, error) {
	if root() != "" {
		override := filepath.Join(root(), OverrideDir, filepath.FromSlash(relPath))
		if data, err := os.ReadFile(override); err == nil {
			logging.DebugInfo(1, "Using override", override)
			return data, nil
		} else if !os.IsNotExist(err) {
			// The file exists but could not be read: that is worth knowing
			// about, since the operator clearly meant to override something.
			logging.Warn("Cannot read resource override", override+":", err)
		}
	}

	return FS.ReadFile(relPath)
}

// Exists reports whether a built-in resource (or an override) is present.
func Exists(relPath string) bool {
	if root() != "" {
		override := filepath.Join(root(), OverrideDir, filepath.FromSlash(relPath))
		if _, err := os.Stat(override); err == nil {
			return true
		}
	}

	_, err := fs.Stat(FS, relPath)
	return err == nil
}
