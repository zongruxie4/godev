package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tinywasm/app"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/kvdb"
)

// TestServerEventDetectionIntegration verifies server file events trigger reloads. Skipped in -short.
func TestServerEventDetectionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - skipping in short mode")
	}

	tmp := t.TempDir()

	// Create proper directory structure using Config methods (type-safe)
	cfg := app.NewConfig(tmp, func(message ...any) {})
	appServerDir := filepath.Join(tmp, cfg.CmdAppServerDir())
	webPublicDir := filepath.Join(tmp, cfg.WebPublicDir())
	require.NoError(t, os.MkdirAll(appServerDir, 0755))
	require.NoError(t, os.MkdirAll(webPublicDir, 0755))

	goModContent := `module testproject

go 1.20
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	serverFilePath := filepath.Join(appServerDir, "server.go")
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
    http.ListenAndServe(":8081", nil)
}
`

	require.NoError(t, os.WriteFile(serverFilePath, []byte(initialServerContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(webPublicDir, "index.html"), []byte("<html>Test</html>"), 0644))

	logs := &SafeBuffer{}
	logger := logs.Log

	// Set up Mock Browser injection
	mockBrowser := &MockBrowser{}

	ExitChan := make(chan bool)
	mockDB, _ := kvdb.New(filepath.Join(tmp, ".env"), logger, app.NewMemoryStore())
	go app.Start(tmp, logger, newUiMockTest(logger), mockBrowser, mockDB, ExitChan, devflow.NewMockGitHubAuth())

	time.Sleep(200 * time.Millisecond)

	// Wait for initialization
	h := app.WaitForActiveHandler(8 * time.Second)
	require.NotNil(t, h)
	// Enable External Server Mode to support reloading on file changes
	require.NoError(t, h.ServerHandler.SetExternalServerMode(true))

	Watcher := app.WaitWatcherReady(8 * time.Second)
	require.NotNil(t, Watcher)

	time.Sleep(500 * time.Millisecond)

	initialReloadCount := int64(mockBrowser.GetReloadCalls())

	modifiedContent := strings.Replace(initialServerContent, "Server v1", "Server v2 MODIFIED", -1)
	modifiedContent = strings.Replace(modifiedContent, "server v1", "server v2 MODIFIED", -1)
	require.NoError(t, os.WriteFile(serverFilePath, []byte(modifiedContent), 0644))

	// Wait for event with timeout
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if int64(mockBrowser.GetReloadCalls()) > initialReloadCount {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	finalReloadCount := int64(mockBrowser.GetReloadCalls())
	reloadDiff := finalReloadCount - initialReloadCount

	close(ExitChan)
	app.SetActiveHandler(nil)

	if reloadDiff == 0 {
		t.Logf("No reloads detected; logs: %v", logs.Lines())
		t.Fail()
	}
}
