package app

import (
	"fmt"
	twfmt "github.com/tinywasm/fmt"
	"os"
	"time"
)

// Logger handles logging operations
type Logger struct {
	RootDir     string
	debug       bool
	initialized bool
	Redir       func(messages ...any)
}

// NewLogger creates a new Logger instance
func NewLogger() *Logger {
	return &Logger{}
}

// SetRootDir sets the root directory for log file
func (l *Logger) SetRootDir(path string) {
	l.RootDir = path
	l.initialized = true
}

// SetDebug enables or disables debug mode
func (l *Logger) SetDebug(v bool) {
	l.debug = v
}

// Logger sends messages through the normal channel (TUI/SSE). Never writes to file.
func (l *Logger) Logger(messages ...any) {
	// Don't write logs if project not initialized
	if !l.initialized {
		return
	}
	// Normal logs flow exclusively through TUI/SSE.
	if l.Redir != nil {
		l.Redir(messages...)
	}
}

func (l *Logger) sprint(messages ...any) string {
	res := ""
	for i, m := range messages {
		if i > 0 {
			res += " "
		}
		res += twfmt.Sprint(m)
	}
	return res
}

// InternalError writes to logs.log only when debug mode is active.
// Use only for errors internal to tinywasm itself, not for project build errors.
func (l *Logger) InternalError(messages ...any) {
	if !l.initialized || !l.debug {
		return
	}

	logPath := "logs.log"
	if l.RootDir != "" {
		logPath = l.RootDir + "/logs.log"
	}

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer logFile.Close()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logFile.WriteString(fmt.Sprintf("[%s] %v\n", timestamp, l.sprint(messages...)))
	}
}
