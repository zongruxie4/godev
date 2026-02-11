package test

import (
	"os"
	"path/filepath"
	"testing"
)

// TestModeSwitchPreservesStateOnCompilationFailure verifies that when a mode change
// triggers a compilation failure, the server does NOT restart and the browser does NOT
// reload. Restarting with the new mode's wasm_exec.js while still serving the old mode's
// .wasm binary causes a WASM runtime mismatch that freezes the browser.
//
// Regression: before the fix, OnWasmExecChange was always called even when compilation
// failed, which caused the browser to reload into a broken state.
func TestModeSwitchPreservesStateOnCompilationFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// No real WASM source file → compilation will fail → compilationSuccess = false
	if err := os.MkdirAll(filepath.Join(tmpDir, "cmd", "web", "client"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	h := NewTestHandler(tmpDir)
	mockBrowser := &MockBrowser{}
	h.Browser = mockBrowser
	h.Tui = newUiMockTest()
	h.GitHandler = &MockGitClient{}
	h.Logger = func(messages ...any) {}
	h.GoModHandler = &MockGoModHandler{}
	h.DB = &MockDB{data: map[string]string{"wasmsize_mode": "L"}}
	h.DevMode = true

	h.InitBuildHandlers()

	before := mockBrowser.GetReloadCalls()

	// Change("L") will fail to compile (no client.go source).
	// compilationSuccess = false → OnWasmExecChange must NOT be called.
	// Without fix: OnWasmExecChange IS called → server restarts with wrong wasm_exec.js → frozen browser.
	h.WasmClient.Change("L")

	if mockBrowser.GetReloadCalls() > before {
		t.Fatalf("Browser.Reload() was called after a failed compilation — "+
			"this causes wasm_exec.js / .wasm mismatch and freezes the browser. "+
			"before=%d after=%d", before, mockBrowser.GetReloadCalls())
	}
}

// TestModeSwitchTriggersBrowserReloadOnSuccess verifies that when a mode change
// compilation succeeds, the server restarts and the browser reloads.
func TestModeSwitchTriggersBrowserReloadOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()

	// Minimal valid WASM project: package main with empty main()
	// WasmClient uses SourceDir="web" + mainInputFile="client.go" → web/client.go
	webDir := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatal(err)
	}
	clientGo := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(webDir, "client.go"), []byte(clientGo), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	h := NewTestHandler(tmpDir)
	mockBrowser := &MockBrowser{}
	h.Browser = mockBrowser
	h.Tui = newUiMockTest()
	h.GitHandler = &MockGitClient{}
	h.Logger = func(messages ...any) {}
	h.GoModHandler = &MockGoModHandler{}
	h.DB = &MockDB{data: map[string]string{"wasmsize_mode": "L"}}
	h.DevMode = true

	h.InitBuildHandlers()
	// Match OnProjectReady: set correct root so compilation resolves go.mod correctly
	h.WasmClient.SetAppRootDir(tmpDir)

	before := mockBrowser.GetReloadCalls()

	// Compilation for mode L (Go standard WASM) should succeed on this minimal project.
	// compilationSuccess = true → OnWasmExecChange fires → Browser.Reload() is called.
	h.WasmClient.Change("L")

	if mockBrowser.GetReloadCalls() <= before {
		t.Fatalf("Browser.Reload() was not called after a successful mode change. before=%d after=%d",
			before, mockBrowser.GetReloadCalls())
	}
}
