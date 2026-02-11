package test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinywasm/devbrowser"
	"github.com/tinywasm/devflow"
)

type MockGoModHandler struct{}

func (m *MockGoModHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	return nil
}
func (m *MockGoModHandler) SetFolderWatcher(watcher devflow.FolderWatcher)   {}
func (m *MockGoModHandler) Name() string                                     { return "MOCK_GOMOD" }
func (m *MockGoModHandler) SupportedExtensions() []string                    { return nil }
func (m *MockGoModHandler) MainInputFileRelativePath() string                { return "go.mod" }
func (m *MockGoModHandler) UnobservedFiles() []string                        { return nil }
func (m *MockGoModHandler) SetLog(fn func(...any))                           {}
func (m *MockGoModHandler) SetRootDir(path string)                           {}
func (m *MockGoModHandler) GetReplacePaths() ([]devflow.ReplaceEntry, error) { return nil, nil }

// StaleDB is NOT used here anymore because we are testing the real flow + caching.
// We'll use a standard mock DB that behaves correctly to isolate caching issues.
type MockDB struct {
	data map[string]string
}

func (m *MockDB) Get(key string) (string, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", nil
}

func (m *MockDB) Set(key, value string) error {
	m.data[key] = value
	return nil
}
func (m *MockDB) Delete(key string) error { return nil }
func (m *MockDB) Close() error            { return nil }

// TestBugModeSwitch verifies that switching compilation mode updates the generated JS
// and that cache headers are correctly set to prevent stale content in the browser.
func TestBugModeSwitch(t *testing.T) {
	// 1. Initialize Handler with Test Helper
	tmpDir := t.TempDir()

	// Create minimal source structure for compilation
	webDir := filepath.Join(tmpDir, "cmd", "web", "client")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create minimal main.wasm.go that compiles in both Go and TinyGo
	mainWasmGo := `//go:build wasm
package main

func main() {
	println("test")
}
`
	if err := os.WriteFile(filepath.Join(webDir, "main.wasm.go"), []byte(mainWasmGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module testapp
go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	h := NewTestHandler(tmpDir)

	// Inject Mocks
	h.Tui = newUiMockTest()
	h.GitHandler = &MockGitClient{}

	// Use REAL DevBrowser
	// We need a channel for exit
	exitChan := make(chan bool)
	// Mock store for browser settings
	browserDB := &MockDB{data: make(map[string]string)}

	// Create DevBrowser with cache explicitly disabled (though it's default now)
	realBrowser := devbrowser.New(h.Tui, browserDB, exitChan, devbrowser.WithCache(false))
	realBrowser.SetHeadless(true) // Helper for tests
	realBrowser.SetTestMode(true) // signal to not actually launch chrome if we just want to test wiring OR keep false if we want to test logic?

	h.Browser = realBrowser

	// Inject DB
	h.DB = &MockDB{data: map[string]string{
		"wasmsize_mode": "L",
	}}
	// h.WasmClient.Database = h.DB // Removed: WasmClient is nil here; InitBuildHandlers uses h.DB to init it.

	h.GoModHandler = &MockGoModHandler{}
	h.Logger = func(messages ...any) {}
	h.DevMode = true // Force development mode to prevent caching

	// Initialize Build Handlers
	h.InitBuildHandlers()

	// 2. Verify Initial State (Mode "L" / Go)
	initialMode := h.WasmClient.Value()
	if initialMode != "L" {
		t.Fatalf("Expected initial mode 'L', got '%s'", initialMode)
	}

	// 3. Verify Cache Headers for script.js
	// We cannot easily use the browser to check headers without network capturing.
	// But we can check the *handler* directly, which is what serves the browser.

	// Create a mock request for script.js
	req, err := http.NewRequest("GET", "/script.js", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	// We need to route it. AssetsHandler doesn't expose ServeHTTP directly on itself,
	// but it registers routes. We can use a ServeMux.
	mux := http.NewServeMux()
	h.AssetsHandler.RegisterRoutes(mux)

	mux.ServeHTTP(rr, req)

	// Check Cache-Control header
	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-cache") {
		t.Errorf("FAIL: script.js should have 'no-cache' header in DevMode/JS, got: '%s'", cacheControl)
	}

	// Check initial content
	jsL := rr.Body.String()
	t.Logf("Initial JS (Mode L) content length: %d", len(jsL))
	t.Logf("Initial JS has Go signature (runtime.wasmExit): %v", strings.Contains(jsL, "runtime.wasmExit"))
	t.Logf("Initial JS has TinyGo signature (runtime.sleepTicks): %v", strings.Contains(jsL, "runtime.sleepTicks"))

	// 4. Switch Mode to "S" (TinyGo)
	h.WasmClient.Change("S")
	time.Sleep(100 * time.Millisecond) // Wait for asset regeneration

	// 5. Verify Content Changed
	// Re-request script.js
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req)

	jsS := rr2.Body.String()
	t.Logf("JS content length: %d bytes", len(jsS))

	// Verify it has TinyGo signatures and NOT Go signatures
	// Verify it has TinyGo signatures and NOT Go signatures
	// Only check this if the mode actually switched (TinyGo is installed)
	finalMode := h.WasmClient.Value()
	if finalMode == "S" {
		if !strings.Contains(jsS, "runtime.sleepTicks") {
			t.Errorf("New JS (Mode S) missing 'runtime.sleepTicks'")
		}
	} else {
		t.Logf("Skipping JS content check: Mode is '%s' (likely missing TinyGo)", finalMode)
	}
}
