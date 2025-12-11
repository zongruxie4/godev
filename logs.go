package tinywasm

import (
	"fmt"
	"os"
	"time"
)

// Logger handles logging operations
type Logger struct{}

// NewLogger creates a new Logger instance
func NewLogger() *Logger {
	return &Logger{}
}

// Logger writes messages to a log file
func (l *Logger) Logger(messages ...any) {
	logFile, err := os.OpenFile("logs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer logFile.Close()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logFile.WriteString(fmt.Sprintf("[%s] %v\n", timestamp, fmt.Sprint(messages...)))
	}
}
