package godev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cdvelop/devwatch"
	"github.com/cdvelop/godepfind"
	"github.com/stretchr/testify/require"
)

// TestGoHandlerRoutingIssue is a unit test that reproduces the specific issue
// where WASM handler without main.wasm.go incorrectly claims database/db.go
// when it should belong to the server handler.
//
// This test directly uses the DevWatch component to isolate the routing issue
// without the full godev application startup overhead.
func TestGoHandlerRoutingIssue(t *testing.T) {
	// Setup temporary project layout that mirrors the real scenario
	tmp := t.TempDir()

	// Create directory structure
	serverDir := filepath.Join(tmp, "pwa")
	databaseDir := filepath.Join(tmp, "database")

	require.NoError(t, os.MkdirAll(serverDir, 0755))
	require.NoError(t, os.MkdirAll(databaseDir, 0755))

	// Create go.mod
	goModContent := `module testproject

go 1.21
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	// Create main.server.go WITH database import
	serverMainPath := filepath.Join(serverDir, "main.server.go")
	serverContent := `package main

import "testproject/database"

func main() {

	database.Connect()

	printf("Server starting on port 4430")
}
`
	require.NoError(t, os.WriteFile(serverMainPath, []byte(serverContent), 0644))

	// Create database/db.go
	dbPath := filepath.Join(databaseDir, "db.go")
	dbContent := `package database

func Connect() {
	println("Connected to database...")
}
`
	require.NoError(t, os.WriteFile(dbPath, []byte(dbContent), 0644))

	// Create tracking handlers to capture which handler claims db.go
	var serverCalls []string
	var wasmCalls []string

	// Create server handler (like goserver.New would do)
	serverHandler := &TestServerHandler{
		mainPath: "pwa/main.server.go",
		calls:    &serverCalls,
	}

	// Create WASM handler WITHOUT main.wasm.go (this simulates the issue)
	wasmHandler := &TestWasmHandler{
		mainPath: "pwa/main.wasm.go", // Note: this file doesn't exist
		calls:    &wasmCalls,
	}

	// Create DevWatch with both handlers
	logOutput := &strings.Builder{}
	watcher := devwatch.New(&devwatch.WatchConfig{
		AppRootDir:      tmp,
		FileEventAssets: &NoOpAssetHandler{}, // Not relevant for this test
		FilesEventGO:    []devwatch.GoFileHandler{serverHandler, wasmHandler},
		FolderEvents:    &NoOpFolderHandler{}, // Not relevant for this test
		BrowserReload:   func() error { return nil },
		Logger:          logOutput,
		ExitChan:        make(chan bool, 1),
	})

	// Simulate file event on database/db.go (this is where the routing issue occurs)
	// We'll simulate this by directly calling the internal logic instead of
	// setting up the full file watching system

	// Get the dependency finder that DevWatch uses internally
	depFinder := godepfind.New(tmp)

	// Test the routing logic: which handler should claim database/db.go?
	t.Logf("Testing file ownership detection for database/db.go")

	// Check server handler
	serverShouldClaim, err := depFinder.ThisFileIsMine(serverHandler.MainInputFileRelativePath(), dbPath, "write")
	require.NoError(t, err)
	t.Logf("Server handler (main: %s) claims db.go: %v", serverHandler.MainInputFileRelativePath(), serverShouldClaim)

	// Check WASM handler (this should fail since main.wasm.go doesn't exist)
	wasmShouldClaim, err := depFinder.ThisFileIsMine(wasmHandler.MainInputFileRelativePath(), dbPath, "write")
	if err != nil {
		t.Logf("WASM handler (main: %s) error claiming db.go: %v", wasmHandler.MainInputFileRelativePath(), err)
		wasmShouldClaim = false
	} else {
		t.Logf("WASM handler (main: %s) claims db.go: %v", wasmHandler.MainInputFileRelativePath(), wasmShouldClaim)
	}

	// Analyze the results
	if wasmShouldClaim && !serverShouldClaim {
		t.Errorf("ISSUE REPRODUCED: WASM handler incorrectly claims database/db.go")
		t.Errorf("WASM handler main file: %s (does not exist)", wasmHandler.MainInputFileRelativePath())
		t.Errorf("Server handler main file: %s (imports testproject/database)", serverHandler.MainInputFileRelativePath())
		t.Errorf("Expected: Only server handler should claim db.go")
		t.Errorf("Actual: WASM handler claims db.go despite missing main.wasm.go")
	} else if wasmShouldClaim && serverShouldClaim {
		t.Errorf("ISSUE DETECTED: Both handlers claim database/db.go")
		t.Errorf("Expected: Only server handler should claim db.go")
		t.Errorf("Actual: Both handlers claim it")
	} else if serverShouldClaim && !wasmShouldClaim {
		t.Logf("SUCCESS: Only server handler correctly claims db.go")
	} else {
		t.Errorf("UNEXPECTED: Neither handler claims db.go")
		t.Errorf("Expected: Server handler should claim db.go (main.server.go imports database)")
		t.Errorf("Actual: Neither handler claims it")
	}

	// Clean up
	watcher.ExitChan <- true
}

// TestServerHandler simulates goserver.ServerHandler for testing
type TestServerHandler struct {
	mainPath string
	calls    *[]string
}

func (h *TestServerHandler) MainInputFileRelativePath() string {
	return h.mainPath
}

func (h *TestServerHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	call := "SERVER: " + event + " " + fileName + " " + filePath
	*h.calls = append(*h.calls, call)
	return nil
}

// TestWasmHandler simulates tinywasm.TinyWasm for testing
type TestWasmHandler struct {
	mainPath string
	calls    *[]string
}

func (h *TestWasmHandler) MainInputFileRelativePath() string {
	return h.mainPath
}

func (h *TestWasmHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	call := "WASM: " + event + " " + fileName + " " + filePath
	*h.calls = append(*h.calls, call)
	return nil
}

// NoOpAssetHandler for testing (not relevant to this test)
type NoOpAssetHandler struct{}

func (h *NoOpAssetHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	return nil
}

// NoOpFolderHandler for testing (not relevant to this test)
type NoOpFolderHandler struct{}

func (h *NoOpFolderHandler) NewFolderEvent(folderName, path, event string) error {
	return nil
}
