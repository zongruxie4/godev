package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinywasm/app"
)

func TestLoggerNoFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tinywasm_logs_test")
	defer os.RemoveAll(tmpDir)

	logger := app.NewLogger()
	logger.SetRootDir(tmpDir)

	logger.Logger("This should not be in a file")

	logPath := filepath.Join(tmpDir, "logs.log")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("logs.log should not exist")
	}
}

func TestInternalError(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tinywasm_internal_test")
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "logs.log")

	// 1. debug=false
	logger := app.NewLogger()
	logger.SetRootDir(tmpDir)
	logger.SetDebug(false)
	logger.InternalError("Debug is false")

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("logs.log should not exist when debug is false")
	}

	// 2. debug=true
	logger.SetDebug(true)
	msg := "Debug is true"
	logger.InternalError(msg)

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), msg) {
		t.Errorf("Log file should contain: %s", msg)
	}
}
