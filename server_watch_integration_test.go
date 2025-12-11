package tinywasm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestServerWatchIntegration reproduces server watch behavior; skipped in -short.
func TestServerWatchIntegration(t *testing.T) {
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

	var logs bytes.Buffer
	logger := func(messages ...any) {
		var msg string
		for i, m := range messages {
			if i > 0 {
				msg += " "
			}
			msg += fmt.Sprint(m)
		}
		logs.WriteString(msg + "\n")
	}

	var reloadCount int64
	var serverRestartEvents []string

	exitChan := make(chan bool)
	go Start(tmp, logger, newUiMockTest(logger), exitChan)

	time.Sleep(100 * time.Millisecond)

	waitStart := time.Now().Add(5 * time.Second)
	for ActiveHandler == nil && time.Now().Before(waitStart) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler)

	SetWatcherBrowserReload(func() error {
		count := atomic.AddInt64(&reloadCount, 1)
		event := fmt.Sprintf("BrowserReload called at %s (count: %d)", time.Now().Format("15:04:05.000"), count)
		serverRestartEvents = append(serverRestartEvents, event)
		return nil
	})

	waitWatcher := time.Now().Add(8 * time.Second)
	for ActiveHandler.watcher == nil && time.Now().Before(waitWatcher) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler.watcher)

	time.Sleep(500 * time.Millisecond)

	initialReloadCount := atomic.LoadInt64(&reloadCount)

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

	time.Sleep(1 * time.Second)

	finalReloadCount := atomic.LoadInt64(&reloadCount)
	reloadDiff := finalReloadCount - initialReloadCount

	exitChan <- true

	if reloadDiff == 0 {
		t.Fatalf("PROBLEM: Server file modifications did not trigger any reloads")
	}
}
