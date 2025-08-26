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

// TestServerEventDetection tests if server file events are properly detected and handled
func TestServerEventDetection(t *testing.T) {
	tmp := t.TempDir()

	// Create the exact structure needed for server detection
	pwaDir := filepath.Join(tmp, "pwa")
	require.NoError(t, os.MkdirAll(pwaDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(pwaDir, "public"), 0755))

	// Create go.mod to establish module context
	goModContent := `module testproject

go 1.20
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	// Create main.server.go file
	serverFilePath := filepath.Join(pwaDir, "main.server.go")
	initialServerContent := `package main

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

	require.NoError(t, os.WriteFile(serverFilePath, []byte(initialServerContent), 0644))

	// Create a static file
	require.NoError(t, os.WriteFile(filepath.Join(pwaDir, "public", "index.html"), []byte("<html>Test</html>"), 0644))

	// Capture all logs to analyze what's happening
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
	}

	// Track events and reloads
	var reloadCount int64
	var eventMessages []string

	// Start godev
	exitChan := make(chan bool)
	go Start(tmp, logger, exitChan)

	// Wait for initialization
	time.Sleep(200 * time.Millisecond)

	// Set up enhanced event tracking
	SetWatcherBrowserReload(func() error {
		count := atomic.AddInt64(&reloadCount, 1)
		timestamp := time.Now().Format("15:04:05.000")
		event := fmt.Sprintf("[%s] BrowserReload called (count: %d)", timestamp, count)
		eventMessages = append(eventMessages, event)
		t.Logf("*** %s ***", event)
		return nil
	})

	// Wait for full initialization
	waitTimeout := time.Now().Add(8 * time.Second)
	for ActiveHandler == nil && time.Now().Before(waitTimeout) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler, "ActiveHandler should be set")

	// Wait for watcher
	for ActiveHandler.watcher == nil && time.Now().Before(waitTimeout) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler.watcher, "Watcher should be created")

	// Additional stabilization time
	time.Sleep(500 * time.Millisecond)

	t.Logf("=== INITIAL STATE ===")
	t.Logf("Project root: %s", tmp)
	t.Logf("Server file: %s", serverFilePath)

	if ActiveHandler.serverHandler != nil {
		t.Logf("ServerHandler.MainFilePath(): %s", ActiveHandler.serverHandler.MainFilePath())
		t.Logf("ServerHandler.Name(): %s", ActiveHandler.serverHandler.Name())
	}

	if ActiveHandler.watcher != nil {
		t.Logf("Watcher FilesEventGO count: %d", len(ActiveHandler.watcher.FilesEventGO))
		for i, handler := range ActiveHandler.watcher.FilesEventGO {
			t.Logf("  Handler %d: %s -> %s", i, handler.Name(), handler.MainFilePath())
		}
	}

	initialReloadCount := atomic.LoadInt64(&reloadCount)
	initialLogCount := len(allLogMessages)
	t.Logf("Initial reload count: %d", initialReloadCount)
	t.Logf("Initial log count: %d", initialLogCount)

	// Modify the server file
	t.Logf("=== MODIFYING SERVER FILE ===")
	modifiedContent := strings.Replace(initialServerContent, "Server v1", "Server v2 MODIFIED", -1)
	modifiedContent = strings.Replace(modifiedContent, "server v1", "server v2 MODIFIED", -1)

	t.Logf("Writing modified content to: %s", serverFilePath)
	require.NoError(t, os.WriteFile(serverFilePath, []byte(modifiedContent), 0644))

	// Wait and monitor for events
	t.Logf("Waiting for event processing...")
	time.Sleep(2 * time.Second)

	finalReloadCount := atomic.LoadInt64(&reloadCount)
	finalLogCount := len(allLogMessages)
	reloadDiff := finalReloadCount - initialReloadCount
	newLogCount := finalLogCount - initialLogCount

	t.Logf("=== RESULTS ===")
	t.Logf("Reload count change: %d -> %d (diff: %d)", initialReloadCount, finalReloadCount, reloadDiff)
	t.Logf("New log messages: %d", newLogCount)
	t.Logf("Event messages: %v", eventMessages)

	// Analyze logs for file events
	t.Logf("=== LOG ANALYSIS ===")
	fileEventLogs := []string{}
	serverEventLogs := []string{}
	watchLogs := []string{}
	errorLogs := []string{}

	for _, logMsg := range allLogMessages[initialLogCount:] {
		lower := strings.ToLower(logMsg)
		if strings.Contains(lower, "file event") || strings.Contains(lower, "event:") {
			fileEventLogs = append(fileEventLogs, logMsg)
		}
		if strings.Contains(lower, "server") && (strings.Contains(lower, "restart") || strings.Contains(lower, "recompil")) {
			serverEventLogs = append(serverEventLogs, logMsg)
		}
		if strings.Contains(lower, "watch") {
			watchLogs = append(watchLogs, logMsg)
		}
		if strings.Contains(lower, "error") {
			errorLogs = append(errorLogs, logMsg)
		}
	}

	t.Logf("File event logs (%d):", len(fileEventLogs))
	for _, log := range fileEventLogs {
		t.Logf("  %s", log)
	}

	t.Logf("Server event logs (%d):", len(serverEventLogs))
	for _, log := range serverEventLogs {
		t.Logf("  %s", log)
	}

	t.Logf("Watch logs (%d):", len(watchLogs))
	for _, log := range watchLogs {
		t.Logf("  %s", log)
	}

	t.Logf("Error logs (%d):", len(errorLogs))
	for _, log := range errorLogs {
		t.Logf("  %s", log)
	}

	// Stop the application
	exitChan <- true

	// Final analysis
	if reloadDiff == 0 {
		t.Logf("❌ PROBLEM: Server file modification did not trigger browser reload")
		t.Logf("This suggests the issue is in:")
		if len(fileEventLogs) == 0 {
			t.Logf("  1. File system events are not being detected")
		} else {
			t.Logf("  1. File events detected (%d) but not processed correctly", len(fileEventLogs))
		}
		if len(serverEventLogs) == 0 {
			t.Logf("  2. Server restart logic is not being triggered")
		}
		if len(errorLogs) > 0 {
			t.Logf("  3. There are errors preventing proper processing")
		}
	} else {
		t.Logf("✅ SUCCESS: Server file modification triggered %d browser reloads", reloadDiff)
	}

	// Show a sample of all logs for debugging
	t.Logf("=== SAMPLE LOGS (last 20) ===")
	startIdx := len(allLogMessages) - 20
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(allLogMessages); i++ {
		t.Logf("%3d: %s", i, allLogMessages[i])
	}
}
