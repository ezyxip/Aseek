package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

type Logger struct {
	mu      sync.Mutex
	w       io.Writer
	level   Level
	module  string
	private bool
}

func New(out io.Writer, level Level, private bool) *Logger {
	return &Logger{w: out, level: level, module: "main", private: private}
}

func Default() *Logger {
	return New(os.Stderr, LevelInfo, false)
}

func (l *Logger) WithModule(module string) *Logger {
	return &Logger{
		w:       l.w,
		level:   l.level,
		module:  module,
		private: l.private,
	}
}

func (l *Logger) log(level Level, msg string, fields ...any) {
	if level < l.level {
		return
	}

	entry := map[string]interface{}{
		"ts":     time.Now().UTC().Format(time.RFC3339Nano),
		"level":  level.String(),
		"module": l.module,
		"msg":    msg,
	}

	if l.private {
		for k := range entry {
			if k != "ts" && k != "level" && k != "module" {
				entry[k] = "[redacted]"
			}
		}
	}

	for i := 0; i < len(fields)-1; i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		entry[key] = fields[i+1]
	}

	data, _ := json.Marshal(entry)

	l.mu.Lock()
	fmt.Fprintln(l.w, string(data))
	l.mu.Unlock()
}

func (l *Logger) Debug(msg string, fields ...any) { l.log(LevelDebug, msg, fields...) }
func (l *Logger) Info(msg string, fields ...any)  { l.log(LevelInfo, msg, fields...) }
func (l *Logger) Warn(msg string, fields ...any)  { l.log(LevelWarn, msg, fields...) }
func (l *Logger) Error(msg string, fields ...any) { l.log(LevelError, msg, fields...) }