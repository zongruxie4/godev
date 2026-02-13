package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tinywasm/app"
)

// TestServerWatchIntegration reproduces server watch behavior; skipped in -short.
func TestServerWatchIntegration(t *testing.T) {
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
    "log"
    "net/http"
    "os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "6060"
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/h", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintln(w, "Server is running v1")
    })

    server := &http.Server{
        Addr:    ":" + port,
        Handler: mux,
    }

    fmt.Printf("Server running on port %s — v1\n", port)

    if err := server.ListenAndServe(); err != nil {
        log.Fatal("Server failed to start:", err)
    }
}
`

	require.NoError(t, os.WriteFile(serverFilePath, []byte(initialServerContent), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(webPublicDir, "index.html"), []byte("<html>Test</html>"), 0644))

	ctx := startTestApp(t, tmp)
	defer ctx.Cleanup()

	h := ctx.Handler

	time.Sleep(200 * time.Millisecond)

	// Enable External Server Mode to support reloading on file changes
	require.NoError(t, h.Server.(*TestServerWrapper).SetExternalServerMode(true))

	Watcher := app.WaitWatcherReady(8 * time.Second)
	require.NotNil(t, Watcher)

	time.Sleep(500 * time.Millisecond)

	initialReloadCount := int64(ctx.Browser.GetReloadCalls())

	modifiedServerContent := `package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "6060"
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/h", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintln(w, "Server is running v2 - MODIFIED")
    })

    server := &http.Server{
        Addr:    ":" + port,
        Handler: mux,
    }

    fmt.Printf("Server running on port %s — v2 MODIFIED\n", port)

    if err := server.ListenAndServe(); err != nil {
        log.Fatal("Server failed to start:", err)
    }
}
`

	require.NoError(t, os.WriteFile(serverFilePath, []byte(modifiedServerContent), 0644))

	// Wait for event with timeout
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if int64(ctx.Browser.GetReloadCalls()) > initialReloadCount {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	finalReloadCount := int64(ctx.Browser.GetReloadCalls())
	reloadDiff := finalReloadCount - initialReloadCount

	if reloadDiff == 0 {
		t.Fatalf("PROBLEM: Server file modifications did not trigger any reloads")
	}
}
