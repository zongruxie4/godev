package app

import (
	"fmt"
	"os"
	"time"
)

// Logger handles logging operations
type Logger struct {
	rootDir     string
	initialized bool
}

// NewLogger creates a new Logger instance
func NewLogger() *Logger {
	return &Logger{}
}

// SetRootDir sets the root directory for log file
func (l *Logger) SetRootDir(path string) {
	l.rootDir = path
	l.initialized = true
}

// Logger writes messages to a log file only if initialized
func (l *Logger) Logger(messages ...any) {
	// Don't write logs if project not initialized
	if !l.initialized {
		return
	}

	logPath := "logs.log"
	if l.rootDir != "" {
		logPath = l.rootDir + "/logs.log"
	}

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer logFile.Close()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logFile.WriteString(fmt.Sprintf("[%s] %v\n", timestamp, fmt.Sprint(messages...)))
	}
}
