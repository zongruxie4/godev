package godev

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWatchEventsDebug specifically tests the watchEvents logic with detailed logging
func TestWatchEventsDebug(t *testing.T) {
	tmp := t.TempDir()

	// Create the exact structure
	pwaDir := filepath.Join(tmp, "pwa")
	require.NoError(t, os.MkdirAll(pwaDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(pwaDir, "public"), 0755))

	// Create go.mod
	goModContent := `module testproject

go 1.20
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	// Create main.server.go file (fix the compile error)
	serverFilePath := filepath.Join(pwaDir, "main.server.go")
	validServerContent := `package main

import (
	"fmt"
	"net/http"
	"log"
)

func main() {
	http.HandleFunc("/h", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Server v1")
	})
	log.Println("Starting server v1")
	http.ListenAndServe(":4430", nil)
}
`

	require.NoError(t, os.WriteFile(serverFilePath, []byte(validServerContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(pwaDir, "public", "index.html"), []byte("<html>Test</html>"), 0644))

	// Enhanced logging to capture everything
	var logs bytes.Buffer
	var allLogMessages []string
	logger := func(messages ...any) {
		var msg string
		for i, m := range messages {
			if i > 0 {
				msg += " "
			}
			msg += fmt.Sprint(m)
		}
		logs.WriteString(msg + "\n")
		allLogMessages = append(allLogMessages, msg)
		// Always log to console for debugging
		t.Logf("LOG: %s", msg)
	}

	var reloadCount int64
	var eventTimestamps []string

	// Start godev
	exitChan := make(chan bool)
	go Start(tmp, logger, exitChan)

	time.Sleep(200 * time.Millisecond)

	SetWatcherBrowserReload(func() error {
		count := atomic.AddInt64(&reloadCount, 1)
		timestamp := time.Now().Format("15:04:05.000")
		event := fmt.Sprintf("[%s] BrowserReload #%d", timestamp, count)
		eventTimestamps = append(eventTimestamps, event)
		t.Logf("*** RELOAD: %s ***", event)
		return nil
	})

	// Wait for complete initialization
	waitTimeout := time.Now().Add(8 * time.Second)
	for ActiveHandler == nil && time.Now().Before(waitTimeout) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler)

	for ActiveHandler.watcher == nil && time.Now().Before(waitTimeout) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler.watcher)

	time.Sleep(500 * time.Millisecond)

	t.Logf("=== SYSTEM STATE ===")
	t.Logf("Project root: %s", tmp)
	t.Logf("Server file: %s", serverFilePath)

	// Log current handler state
	if ActiveHandler.serverHandler != nil {
		mainFilePath := ActiveHandler.serverHandler.MainFilePath()
		t.Logf("ServerHandler.MainFilePath(): %s", mainFilePath)
		t.Logf("ServerHandler.Name(): %s", ActiveHandler.serverHandler.Name())

		// Compare paths
		t.Logf("Expected server path: %s", serverFilePath)
		t.Logf("Handler expects path: %s", mainFilePath)
		t.Logf("Paths match: %v", serverFilePath == mainFilePath)

		// Check relative path from project root
		relServerPath, _ := filepath.Rel(tmp, serverFilePath)
		relHandlerPath, _ := filepath.Rel(tmp, mainFilePath)
		t.Logf("Relative server path: %s", relServerPath)
		t.Logf("Relative handler path: %s", relHandlerPath)
		t.Logf("Relative paths match: %v", relServerPath == relHandlerPath)
	}

	if ActiveHandler.watcher != nil {
		t.Logf("FilesEventGO handlers: %d", len(ActiveHandler.watcher.FilesEventGO))
		for i, handler := range ActiveHandler.watcher.FilesEventGO {
			t.Logf("  [%d] %s -> %s", i, handler.Name(), handler.MainFilePath())
		}
	}

	// Capture current state
	initialReloadCount := atomic.LoadInt64(&reloadCount)
	logCountBefore := len(allLogMessages)

	t.Logf("=== BEFORE MODIFICATION ===")
	t.Logf("Reload count: %d", initialReloadCount)
	t.Logf("Log messages so far: %d", logCountBefore)

	// Modify the server file with a very obvious change
	t.Logf("=== MODIFYING SERVER FILE ===")
	modifiedContent := strings.Replace(validServerContent, "Server v1", "Server v2 MODIFIED", -1)
	modifiedContent = strings.Replace(modifiedContent, "server v1", "server v2 MODIFIED", -1)

	t.Logf("Writing to: %s", serverFilePath)
	err := os.WriteFile(serverFilePath, []byte(modifiedContent), 0644)
	require.NoError(t, err)
	t.Logf("File written successfully at %s", time.Now().Format("15:04:05.000"))

	// Wait and capture what happens
	t.Logf("=== MONITORING EVENTS ===")
	for i := 0; i < 20; i++ { // Monitor for 2 seconds
		time.Sleep(100 * time.Millisecond)
		currentReloadCount := atomic.LoadInt64(&reloadCount)
		currentLogCount := len(allLogMessages)
		newLogs := currentLogCount - logCountBefore

		t.Logf("  [%d] Reloads: %d, New logs: %d", i, currentReloadCount, newLogs)

		if currentReloadCount > initialReloadCount {
			t.Logf("  RELOAD DETECTED at iteration %d!", i)
			break
		}
	}

	finalReloadCount := atomic.LoadInt64(&reloadCount)
	finalLogCount := len(allLogMessages)
	reloadDiff := finalReloadCount - initialReloadCount
	newLogCount := finalLogCount - logCountBefore

	t.Logf("=== FINAL RESULTS ===")
	t.Logf("Reload difference: %d", reloadDiff)
	t.Logf("New log messages: %d", newLogCount)
	t.Logf("Event timestamps: %v", eventTimestamps)

	// Analyze the new log messages
	t.Logf("=== NEW LOG ANALYSIS ===")
	for i, logMsg := range allLogMessages[logCountBefore:] {
		t.Logf("[%d] %s", i+logCountBefore, logMsg)
	}

	// Look for specific patterns in logs
	fileEventFound := false
	serverEventFound := false
	errorFound := false

	for _, logMsg := range allLogMessages[logCountBefore:] {
		lower := strings.ToLower(logMsg)
		if strings.Contains(lower, "file event") || strings.Contains(lower, "main.server.go") {
			fileEventFound = true
			t.Logf("FILE EVENT LOG: %s", logMsg)
		}
		if strings.Contains(lower, "server") && (strings.Contains(lower, "restart") || strings.Contains(lower, "recompil")) {
			serverEventFound = true
			t.Logf("SERVER EVENT LOG: %s", logMsg)
		}
		if strings.Contains(lower, "error") {
			errorFound = true
			t.Logf("ERROR LOG: %s", logMsg)
		}
	}

	// Stop the application
	exitChan <- true

	// Final diagnosis
	t.Logf("=== DIAGNOSIS ===")
	if reloadDiff == 0 {
		t.Logf("❌ NO RELOAD TRIGGERED")
		if !fileEventFound {
			t.Logf("  🔍 No file events detected in logs")
			t.Logf("  📝 This suggests the file watcher is not detecting the change")
		} else {
			t.Logf("  ✅ File events detected but not processed correctly")
		}

		if !serverEventFound {
			t.Logf("  🔍 No server restart events detected")
			t.Logf("  📝 This suggests godepfind.ThisFileIsMine is returning false")
		}

		if errorFound {
			t.Logf("  ⚠️  Errors detected - check error logs above")
		}
	} else {
		t.Logf("✅ SUCCESS: %d reloads triggered", reloadDiff)
	}
}
