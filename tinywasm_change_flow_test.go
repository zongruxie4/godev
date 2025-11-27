package golite

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

// TestTinyWasmChangeFlow verifies that changing TinyWasm mode triggers
// AssetMin update and a SINGLE browser reload.
func TestTinyWasmChangeFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - skipping in short mode")
	}

	tmp := t.TempDir()

	// Create proper directory structure
	cfg := NewConfig(tmp, func(message ...any) {})
	webPublicDir := filepath.Join(tmp, cfg.WebPublicDir())
	cmdWebClientDir := filepath.Join(tmp, cfg.CmdWebClientDir())
	require.NoError(t, os.MkdirAll(webPublicDir, 0755))
	require.NoError(t, os.MkdirAll(cmdWebClientDir, 0755))

	// Create go.mod
	goModContent := `module testproject

go 1.20
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	// Create main.wasm.go to ensure TinyWasm detects it as a WASM project
	mainWasmPath := filepath.Join(cmdWebClientDir, "main.wasm.go")
	mainWasmContent := `package main
func main() {}
`
	require.NoError(t, os.WriteFile(mainWasmPath, []byte(mainWasmContent), 0644))

	var reloadCount int64
	logger := func(messages ...any) {
		// Silent logger for test
	}

	// Start golite
	exitChan := make(chan bool)
	go Start(tmp, logger, newUiMockTest(logger), exitChan)

	// Wait for initialization
	waitTimeout := time.Now().Add(5 * time.Second)
	for ActiveHandler == nil && time.Now().Before(waitTimeout) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler)

	// Wait for watcher
	for ActiveHandler.watcher == nil && time.Now().Before(waitTimeout) {
		time.Sleep(50 * time.Millisecond)
	}
	require.NotNil(t, ActiveHandler.watcher)

	// Set up browser reload tracking
	SetWatcherBrowserReload(func() error {
		atomic.AddInt64(&reloadCount, 1)
		return nil
	})

	time.Sleep(500 * time.Millisecond)

	// Capture initial state
	initialReloadCount := atomic.LoadInt64(&reloadCount)

	// Trigger TinyWasm Change
	// We simulate a user changing the mode via the UI
	// This should trigger:
	// 1. TinyWasm mode update
	// 2. OnWasmExecChange callback
	// 3. AssetMin.UpdateAssetContent("wasm_exec.js")
	// 4. AssetMin writes to disk
	// 5. Watcher detects write (if configured) OR AssetMin triggers reload directly?
	//    Actually, AssetMin writes to disk. The Watcher watches the output folder?
	//    Wait, AssetMin output is in WebPublicDir. Watcher usually ignores output files to avoid loops.
	//    Let's check how reload is triggered.
	//    In `events.go` of `assetmin`, `NewFileEvent` returns nil.
	//    If `UpdateAssetContent` calls `processAndWrite`, it writes to disk.
	//    If the watcher is watching the output file, it might trigger.
	//    BUT `assetmin` usually has `UnobservedFiles` including output files.
	//    So how does the browser reload happen?
	//    In `golite`, `h.watcher` has `FilesEventHandlers` including `h.assetsHandler`.
	//    When `assetmin` receives an event, it processes it.
	//    If `assetmin` modifies a file, does it return an error? No.
	//    The `devwatch` loop calls `ReloadBrowser()` if `ShouldReloadBrowser()` is true.
	//    `assetmin` doesn't seem to explicitly request reload in `NewFileEvent`.
	//    However, `devwatch` reloads if ANY handler processes an event successfully (returns nil error) AND the event was relevant.
	//    Wait, `UpdateAssetContent` is a direct call, NOT a file event from the watcher.
	//    So `devwatch` is NOT involved in the `UpdateAssetContent` call chain directly.
	//    WE NEED TO ENSURE BROWSER RELOAD HAPPENS.
	//    If `UpdateAssetContent` writes to disk, and that file is ignored by watcher, then watcher won't see it.
	//    So `golite` or `tinywasm` callback might need to trigger reload?
	//    OR `assetmin` should trigger it?
	//    Let's verify what happens. If the test fails, we know we missed a link.

	progressChan := make(chan string, 10)
	go func() {
		// Change to "Small" mode (S)
		ActiveHandler.wasmHandler.Change("S", progressChan)
		close(progressChan)
	}()

	// Wait for change to complete
	for msg := range progressChan {
		fmt.Println("TinyWasm Change:", msg)
	}

	// Wait for potential reload
	time.Sleep(1 * time.Second)

	finalReloadCount := atomic.LoadInt64(&reloadCount)
	reloadDiff := finalReloadCount - initialReloadCount

	// Stop application
	exitChan <- true

	// Verify results
	// We expect exactly 1 reload if the flow works correctly
	if reloadDiff != 1 {
		t.Errorf("Expected exactly 1 browser reload, got %d", reloadDiff)
	}

	// Verify wasm_exec.js does NOT exist as a standalone file
	wasmExecPath := filepath.Join(webPublicDir, "wasm_exec.js")
	if _, err := os.Stat(wasmExecPath); !os.IsNotExist(err) {
		t.Errorf("wasm_exec.js SHOULD NOT exist at %s, but it does", wasmExecPath)
	}

	// Verify script.js contains the TinyWasm content
	// The default script name is script.js (from AssetMin defaults)
	scriptJsPath := filepath.Join(webPublicDir, "script.js")
	content, err := os.ReadFile(scriptJsPath)
	if err != nil {
		t.Fatalf("Failed to read script.js at %s: %v", scriptJsPath, err)
	}

	// Check for TinyWasm signature or specific content
	// TinyWasm adds a header like "// TinyWasm: mode=..."
	// Or we can check for "Go" class definition which is in wasm_exec.js
	if !bytes.Contains(content, []byte("TinyWasm")) && !bytes.Contains(content, []byte("class Go")) {
		t.Errorf("script.js does not appear to contain wasm_exec.js content. Size: %d", len(content))
	}
}
