// Package logging is the server's logger.
//
// Every record is written to stdout as a single-line JSON object, one record per
// line, so that the output can be piped straight into a log collector without a
// parsing step:
//
//	{"ts":"2026-10-13T22:35:52.123Z","level":"info","msg":"Server is running at ...","caller":"main.go:292"}
//
// Nothing is written to disk any more; the previous file logger has been
// removed. Redirect stdout if you want a file.
package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level is how much debug output to emit, taken from debug.level in server.yaml:
//
//	0 = no DebugInfo at all
//	1 = also DebugInfo(1, ...)
//	2 = also DebugInfo(2, ...)
//
// It only gates DebugInfo. Info, Warn and Error are always written, so that the
// startup banner (ports, data directory, configuration source) stays visible
// however this is set.
var Level uint32

const (
	levelDebug = "debug"
	levelInfo  = "info"
	levelWarn  = "warn"
	levelError = "error"
)

type record struct {
	Time    string `json:"ts"`
	Level   string `json:"level"`
	Message string `json:"msg"`
	Caller  string `json:"caller,omitempty"`
}

var mu sync.Mutex

// message renders log operands the way the standard library's log.Println does:
// space separated, matching the previous output format.
func message(v ...any) string {
	if len(v) == 1 {
		if s, ok := v[0].(string); ok {
			return s
		}
	}
	return strings.TrimSuffix(fmt.Sprintln(v...), "\n")
}

func caller(skip int) string {
	// skip is counted from this function: 0 = caller, 1 = write.
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	if i := strings.LastIndexAny(file, `/\`); i >= 0 {
		file = file[i+1:]
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// write emits one JSON log line.
//
// The whole record is rendered into a buffer and handed to a single Write call,
// so records from different goroutines cannot interleave.
func write(level string, v ...any) {
	// Frames: 0 = caller, 1 = write, 2 = the exported helper, 3 = the real
	// call site.
	rec := record{
		Time:    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		Level:   level,
		Message: message(v...),
		Caller:  caller(3),
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// Keep paths and punctuation readable: HTML escaping would turn
	// "<zh-cn.ini>" into "\u003czh-cn.ini\u003e".
	enc.SetEscapeHTML(false)

	if err := enc.Encode(rec); err != nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	_, _ = os.Stdout.Write(buf.Bytes())
}

// DebugInfo logs at the given verbosity.
//
// It is kept because the whole code base calls it, with the original semantics:
// the message is emitted only when l <= Level.
func DebugInfo(l uint32, v ...any) {
	if l > Level {
		return
	}
	if l >= 2 {
		write(levelDebug, v...)
		return
	}
	write(levelInfo, v...)
}

// Info logs a message that is always emitted.
func Info(v ...any) { write(levelInfo, v...) }

// Warn logs a warning that is always emitted.
func Warn(v ...any) { write(levelWarn, v...) }

// Error logs an error that is always emitted.
func Error(v ...any) { write(levelError, v...) }
