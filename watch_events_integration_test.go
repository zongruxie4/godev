package tinywasm

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

// TestWatchEventsIntegration tests the watchEvents logic end-to-end. Skipped in -short.
func TestWatchEventsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - skipping in short mode")
	}

	tmp := t.TempDir()

	// Create proper directory structure using Config methods (type-safe)
	cfg := NewConfig(tmp, func(message ...any) {})
	appServerDir := filepath.Join(tmp, cfg.CmdAppServerDir())
	webPublicDir := filepath.Join(tmp, cfg.WebPublicDir())
	require.NoError(t, os.MkdirAll(appServerDir, 0755))
	require.NoError(t, os.MkdirAll(webPublicDir, 0755))

	// Create go.mod
	goModContent := `module testproject

go 1.20
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	// Create main.go file in appserver directory
	serverFilePath := filepath.Join(appServerDir, "main.go")
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
    http.ListenAndServe(":6060", nil)
}
`

	require.NoError(t, os.WriteFile(serverFilePath, []byte(validServerContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(webPublicDir, "index.html"), []byte("<html>Test</html>"), 0644))

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

	var reloadCount int64
	var eventTimestamps []string

	// Start golite
	exitChan := make(chan bool)
	go Start(tmp, logger, newUiMockTest(logger), exitChan)

	time.Sleep(200 * time.Millisecond)

	SetWatcherBrowserReload(func() error {
		count := atomic.AddInt64(&reloadCount, 1)
		timestamp := time.Now().Format("15:04:05.000")
		event := fmt.Sprintf("[%s] BrowserReload #%d", timestamp, count)
		eventTimestamps = append(eventTimestamps, event)
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

	// Capture current state
	initialReloadCount := atomic.LoadInt64(&reloadCount)
	logCountBefore := len(allLogMessages)

	// Modify the server file
	modifiedContent := strings.Replace(validServerContent, "Server v1", "Server v2 MODIFIED", -1)
	modifiedContent = strings.Replace(modifiedContent, "server v1", "server v2 MODIFIED", -1)

	require.NoError(t, os.WriteFile(serverFilePath, []byte(modifiedContent), 0644))

	// Wait and capture what happens (short loop, rely on SetWatcherBrowserReload)
	for i := 0; i < 40; i++ {
		time.Sleep(50 * time.Millisecond)
		if atomic.LoadInt64(&reloadCount) > initialReloadCount {
			break
		}
	}

	finalReloadCount := atomic.LoadInt64(&reloadCount)
	reloadDiff := finalReloadCount - initialReloadCount

	// Stop the application
	exitChan <- true

	if reloadDiff == 0 {
		t.Logf("WARNING: no reloads detected during integration run")
		t.Logf("Logs snapshot (%d): %v", len(allLogMessages)-logCountBefore, allLogMessages[logCountBefore:])
		t.Fail()
	}
}
