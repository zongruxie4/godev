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

// TestExternalServerBrowserOpens replicates the EXACT production bug:
// In site/example (external server mode), the browser never opens or gets ERR_CONNECTION_REFUSED.
// In emptyfolder (internal server mode), the browser opens correctly.
//
// This test MUST FAIL until the bug is fixed.
func TestExternalServerBrowserOpens(t *testing.T) {
	tmp := t.TempDir()

	logs := &SafeBuffer{}
	logger := logs.Log

	// Create an initialized project with go.mod
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testexternalserver\n\ngo 1.21\n"), 0644))

	// Create web directory structure
	cfg := app.NewConfig(tmp, logger)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.WebDir()), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.WebPublicDir()), 0755))

	// Create minimal client.go for WASM
	clientCode := `package main

func main() {}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, cfg.WebDir(), "client.go"), []byte(clientCode), 0644))

	// Create index.html
	require.NoError(t, os.WriteFile(filepath.Join(tmp, cfg.WebPublicDir(), "index.html"), []byte("<html><body>Test</body></html>"), 0644))

	// Create EXTERNAL server file - this triggers external server mode
	serverCode := `package main

import (
	"flag"
	"fmt"
	"net/http"
)

func main() {
	port := flag.String("port", "6060", "server port")
	publicDir := flag.String("public-dir", ".", "public directory")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK from external server")
	})

	fmt.Printf("External server starting on port %s, public=%s\n", *port, *publicDir)
	http.ListenAndServe(":"+*port, nil)
}
`
	// Create the server file in the location that triggers external mode
	serverDir := filepath.Join(tmp, cfg.CmdAppServerDir())
	require.NoError(t, os.MkdirAll(serverDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(serverDir, cfg.ServerFileName()), []byte(serverCode), 0644))

	// Inject MockBrowser that tracks when it actually opens
	mockBrowser := &MockBrowser{}
	mockDB, _ := kvdb.New(filepath.Join(tmp, ".env"), logger, app.NewMemoryStore())

	// IMPORTANT: Disable TestMode so AutoStart actually runs
	originalTestMode := app.TestMode
	app.TestMode = false
	defer func() { app.TestMode = originalTestMode }()

	exitChan := make(chan bool)

	// Start app - should detect external server and switch to external mode
	go app.Start(tmp, logger, newUiMockTest(logger), mockBrowser, mockDB, exitChan, devflow.NewMockGitHubAuth(), &MockGitClient{})

	// Wait for initialization
	h := app.WaitForActiveHandler(10 * time.Second)
	require.NotNil(t, h, "Handler should be initialized")

	// Wait for external server to compile and start + browser to open
	// Give it enough time for compilation (~5-10 seconds)
	time.Sleep(15 * time.Second)

	// Check results
	autoStartCalls := mockBrowser.GetOpenCalls()
	openCalls := mockBrowser.GetOpenCalls()

	logContent := logs.String()
	t.Logf("Open calls: %d", openCalls)
	t.Logf("Logs:\n%s", logContent)

	// Cleanup
	close(exitChan)
	app.SetActiveHandler(nil)
	time.Sleep(500 * time.Millisecond)

	// ASSERTIONS - these should FAIL if bug exists:

	// 1. AutoStart should have been called
	if autoStartCalls == 0 {
		t.Errorf("BUG: AutoStart was never called!")
	}

	// 2. Browser should have opened (openCalls > 0 means serverReady was signaled)
	if openCalls == 0 {
		t.Errorf("BUG: Browser never opened! serverReady channel was never signaled or browser is stuck waiting")
		t.Logf("This replicates the production bug where browser never opens in external server mode")
	}

	// 3. Check for connection refused errors in logs
	if strings.Contains(logContent, "ERR_CONNECTION_REFUSED") {
		t.Errorf("BUG: Browser got ERR_CONNECTION_REFUSED - server wasn't ready when browser tried to connect")
	}

	// 4. Check for the "Server is now accepting connections" log
	if !strings.Contains(logContent, "Server is now accepting connections") {
		t.Logf("WARNING: 'Server is now accepting connections' log not found - port check may not have succeeded")
	}
}

// TestInternalServerBrowserOpens verifies internal server mode works (control test)
func TestInternalServerBrowserOpens(t *testing.T) {
	tmp := t.TempDir()

	logs := &SafeBuffer{}
	logger := logs.Log

	// Create initialized project WITHOUT external server file
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testinternalserver\n\ngo 1.21\n"), 0644))

	cfg := app.NewConfig(tmp, logger)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.WebDir()), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.WebPublicDir()), 0755))

	// Minimal client.go
	clientCode := `package main

func main() {}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, cfg.WebDir(), "client.go"), []byte(clientCode), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, cfg.WebPublicDir(), "index.html"), []byte("<html>Test</html>"), 0644))

	// NO server.go file = internal server mode

	mockBrowser := &MockBrowser{}

	// app.SetInitialBrowser(mockBrowser) // Removed
	// defer app.SetInitialBrowser(nil) // Removed

	originalTestMode := app.TestMode
	app.TestMode = false
	defer func() { app.TestMode = originalTestMode }()

	ExitChan := make(chan bool) // Changed to ExitChan
	mockDB, _ := kvdb.New(filepath.Join(tmp, ".env"), logger, app.NewMemoryStore())
	go app.Start(tmp, logger, newUiMockTest(logger), mockBrowser, mockDB, ExitChan, devflow.NewMockGitHubAuth(), &MockGitClient{}) // Modified app.Start call

	h := app.WaitForActiveHandler(10 * time.Second)
	require.NotNil(t, h, "Handler should be initialized")

	// Wait for browser to open
	time.Sleep(500 * time.Millisecond)
	require.Equal(t, 1, mockBrowser.GetOpenCalls())
	openCalls := mockBrowser.GetOpenCalls()

	logContent := logs.String()
	t.Logf("Open calls: %d", openCalls) // Modified

	close(ExitChan) // Changed to ExitChan
	app.SetActiveHandler(nil)
	time.Sleep(500 * time.Millisecond)

	// Internal server mode should work
	if openCalls == 0 {
		t.Errorf("Browser never opened in internal mode - this indicates a different bug")
		t.Logf("Logs:\n%s", logContent)
	}

	if strings.Contains(logContent, "ERR_CONNECTION_REFUSED") {
		t.Errorf("Internal server got ERR_CONNECTION_REFUSED - should not happen")
	}
}
