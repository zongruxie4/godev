package test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClientGenerationInEmptyFolder(t *testing.T) {
	// 1. Create a temporary directory structure:
	// /start (with go.mod) -> /start/subdir (empty, where we run app)
	tmpRoot := t.TempDir()

	// Create go.mod in root
	if err := os.WriteFile(filepath.Join(tmpRoot, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create and use subdir
	tmpDir := filepath.Join(tmpRoot, "subdir")
	if err := os.Mkdir(tmpDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 2. Start the app in this empty subdirectory
	// startTestApp will set up the environment and start the app
	ctx := startTestApp(t, tmpDir)
	defer ctx.Cleanup()

	// 3. Wait a moment for startup (OnProjectReady to fire + Server Start + Port Probe)
	time.Sleep(500 * time.Millisecond)

	// 4. Check if web/client.go was created
	clientPath := filepath.Join(tmpDir, "web", "client.go")
	if _, err := os.Stat(clientPath); os.IsNotExist(err) {
		entries, _ := os.ReadDir(tmpDir)
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("Expected web/client.go to be created in empty folder, but it was not found.\nDir contents: %v\nLogs:\n%s", names, ctx.Logs.String())
	}

	// 5. Verify the logs indicate generation
	logs := ctx.Logs.String()
	if !strings.Contains(logs, "Generated WASM source file") {
		t.Errorf("Expected logs to mention WASM file generation. Logs:\n%s", logs)
	}

	// 6. Verify that the browser was opened (or attempted to open)
	// In-Memory mode should still trigger browser open once the server is ready
	if ctx.Browser.GetOpenCalls() == 0 {
		t.Errorf("Expected browser to be opened, but GetOpenCalls() is 0. Logs:\n%s", logs)
	}

	// 7. Verify that client.wasm is served (compilation happened)
	// We need the port that the server started on.
	// In the mock context, we don't strictly have the real server object to ask, but we scan logs or use fixed port if mock was set setup that way?
	// startTestApp sets port via env var usually.

	// Wait a bit for server to be fully ready and compile to finish (it happens in goroutine potentially?)
	// Actually generator calls Compile synchronously, but server start is async.
	time.Sleep(100 * time.Millisecond)

	port := os.Getenv("PORT")
	if port == "" {
		port = "6060" // Fallback if env not set correctly?
	}

	resp, err := http.Get("http://localhost:" + port + "/client.wasm")
	if err != nil {
		t.Fatalf("Failed to request client.wasm: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK for client.wasm, got %d", resp.StatusCode)
	}
}
