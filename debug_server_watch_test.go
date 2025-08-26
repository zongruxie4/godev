package godev

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

// TestDebugServerWatch reproduces the issue where Go server files are not properly detected
// and their events don't trigger server restarts
func TestDebugServerWatch(t *testing.T) {
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

	// Create main.server.go file (this is the file we want to watch)
	serverFilePath := filepath.Join(pwaDir, "main.server.go")
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
		port = "4430"
	}

	// Version 1
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

	// Create a static file for the server to serve
	require.NoError(t, os.WriteFile(filepath.Join(pwaDir, "public", "index.html"), []byte("<html>Test</html>"), 0644))

	// Capture all logs for debugging
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
		t.Logf("LOG: %s", msg)
	}

	// Track browser reload calls and server restarts
	var reloadCount int64
	var serverRestartEvents []string

	// Start godev
	exitChan := make(chan bool)
	go Start(tmp, logger, exitChan)

	// Wait for Start to initialize
	time.Sleep(100 * time.Millisecond)

	// Wait for ActiveHandler to be set
	waitStart := time.Now().Add(5 * time.Second)
	for ActiveHandler == nil && time.Now().Before(waitStart) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler, "ActiveHandler should be set after Start()")

	// Set up browser reload tracking to detect server events
	SetWatcherBrowserReload(func() error {
		count := atomic.AddInt64(&reloadCount, 1)
		event := fmt.Sprintf("BrowserReload called at %s (count: %d)", time.Now().Format("15:04:05.000"), count)
		serverRestartEvents = append(serverRestartEvents, event)
		t.Logf("*** %s ***", event)
		return nil
	})

	// Wait for watcher to be fully initialized
	waitWatcher := time.Now().Add(8 * time.Second)
	for ActiveHandler.watcher == nil && time.Now().Before(waitWatcher) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler.watcher, "Watcher should be created")

	// Wait for system to stabilize and initial scan to complete
	time.Sleep(500 * time.Millisecond)

	t.Logf("=== SYSTEM INITIALIZED ===")
	t.Logf("ActiveHandler: %v", ActiveHandler != nil)
	t.Logf("ServerHandler: %v", ActiveHandler.serverHandler != nil)
	if ActiveHandler.serverHandler != nil {
		t.Logf("ServerHandler.MainFilePath(): %s", ActiveHandler.serverHandler.MainFilePath())
		t.Logf("ServerHandler.Name(): %s", ActiveHandler.serverHandler.Name())
	}
	t.Logf("Watcher: %v", ActiveHandler.watcher != nil)
	t.Logf("Project Root: %s", tmp)
	t.Logf("Server File Path: %s", serverFilePath)

	// Debug: Check handler configuration
	if ActiveHandler.watcher != nil && ActiveHandler.serverHandler != nil {
		t.Logf("=== HANDLER DEBUG ===")
		t.Logf("Handler MainFilePath: %s", ActiveHandler.serverHandler.MainFilePath())
		t.Logf("Handler Name: %s", ActiveHandler.serverHandler.Name())
		t.Logf("FilesEventGO handlers count: %d", len(ActiveHandler.watcher.FilesEventGO))
		for i, handler := range ActiveHandler.watcher.FilesEventGO {
			t.Logf("  Handler %d: %s -> %s", i, handler.Name(), handler.MainFilePath())
		}
	}

	initialReloadCount := atomic.LoadInt64(&reloadCount)
	t.Logf("Initial reload count: %d", initialReloadCount)

	// Now modify the server file - this should trigger a server restart
	t.Logf("=== MODIFYING SERVER FILE ===")
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
		port = "4430"
	}

	// Version 2 - MODIFIED
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

	t.Logf("Writing modified content to: %s", serverFilePath)
	require.NoError(t, os.WriteFile(serverFilePath, []byte(modifiedServerContent), 0644))

	// Wait for the file event to be processed
	t.Logf("Waiting for file event processing...")
	time.Sleep(1 * time.Second)

	finalReloadCount := atomic.LoadInt64(&reloadCount)
	reloadDiff := finalReloadCount - initialReloadCount

	t.Logf("=== RESULTS ===")
	t.Logf("Reload count before modification: %d", initialReloadCount)
	t.Logf("Reload count after modification: %d", finalReloadCount)
	t.Logf("Reload difference: %d", reloadDiff)
	t.Logf("Server restart events: %v", serverRestartEvents)

	// Show detailed logs for debugging
	t.Logf("=== FULL LOGS ===")
	t.Logf("%s", logs.String())

	// Test additional modifications to see if the problem persists
	t.Logf("=== SECOND MODIFICATION ===")
	secondModifiedContent := `package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4430"
	}

	// Version 3 - SECOND MODIFICATION
	mux := http.NewServeMux()
	mux.HandleFunc("/h", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Server is running v3 - SECOND MODIFICATION")
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	fmt.Printf("Server running on port %s — v3 SECOND MODIFICATION\n", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
`

	preSecondModificationCount := atomic.LoadInt64(&reloadCount)
	require.NoError(t, os.WriteFile(serverFilePath, []byte(secondModifiedContent), 0644))
	time.Sleep(1 * time.Second)

	postSecondModificationCount := atomic.LoadInt64(&reloadCount)
	secondReloadDiff := postSecondModificationCount - preSecondModificationCount

	t.Logf("Second modification reload difference: %d", secondReloadDiff)

	// Stop the application
	exitChan <- true

	// Analysis: If reloadDiff is 0, the server file changes are not being detected
	if reloadDiff == 0 && secondReloadDiff == 0 {
		t.Errorf("PROBLEM CONFIRMED: Server file modifications did not trigger any reloads")
		t.Errorf("This indicates that either:")
		t.Errorf("1. File events are not reaching watchEvents.go")
		t.Errorf("2. godepfind.ThisFileIsMine is not properly identifying the server file")
		t.Errorf("3. The handler is not being called or is returning an error")
		t.Errorf("4. The server restart is not triggering browser reload")

		// Additional debugging information
		if ActiveHandler != nil && ActiveHandler.watcher != nil && ActiveHandler.serverHandler != nil {
			t.Logf("Debugging information:")
			t.Logf("- Watcher exists: true")
			t.Logf("- ServerHandler exists: true")
			t.Logf("- FilesEventGO length: %d", len(ActiveHandler.watcher.FilesEventGO))
			for i, handler := range ActiveHandler.watcher.FilesEventGO {
				t.Logf("  Handler %d: %s -> %s", i, handler.Name(), handler.MainFilePath())
			}
		}
	} else {
		t.Logf("✓ Server file modifications triggered %d reloads (working correctly)", reloadDiff+secondReloadDiff)
	}
}
