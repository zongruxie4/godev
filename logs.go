package godev

import (
	"fmt"
	"os"
	"time"
)

func (h *handler) LogToFile(messageErr any) {
	logFile, err := os.OpenFile("logs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer logFile.Close()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logFile.WriteString(fmt.Sprintf("[%s] %v\n", timestamp, messageErr))
	}
}
