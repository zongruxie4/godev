package golite

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

// TestServerEventDetectionIntegration verifies server file events trigger reloads. Skipped in -short.
func TestServerEventDetectionIntegration(t *testing.T) {
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

	goModContent := `module testproject

go 1.20
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	serverFilePath := filepath.Join(appServerDir, "main.go")
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
	var eventMessages []string

	exitChan := make(chan bool)
	go Start(tmp, logger, newUiMockTest(logger), exitChan)

	time.Sleep(200 * time.Millisecond)

	SetWatcherBrowserReload(func() error {
		count := atomic.AddInt64(&reloadCount, 1)
		timestamp := time.Now().Format("15:04:05.000")
		event := fmt.Sprintf("[%s] BrowserReload called (count: %d)", timestamp, count)
		eventMessages = append(eventMessages, event)
		return nil
	})

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

	initialReloadCount := atomic.LoadInt64(&reloadCount)

	modifiedContent := strings.Replace(initialServerContent, "Server v1", "Server v2 MODIFIED", -1)
	modifiedContent = strings.Replace(modifiedContent, "server v1", "server v2 MODIFIED", -1)
	require.NoError(t, os.WriteFile(serverFilePath, []byte(modifiedContent), 0644))

	// Wait for event with timeout
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&reloadCount) > initialReloadCount {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	finalReloadCount := atomic.LoadInt64(&reloadCount)
	reloadDiff := finalReloadCount - initialReloadCount

	exitChan <- true

	if reloadDiff == 0 {
		t.Logf("No reloads detected; logs: %v", allLogMessages)
		t.Fail()
	}
}
