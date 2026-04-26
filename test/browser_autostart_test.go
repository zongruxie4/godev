package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinywasm/app"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/kvdb"
)

// TestBrowserAutoStartCalledOnce verifies that when starting the app in an
// initialized project directory, Browser.AutoStart() is called exactly once.
func TestBrowserAutoStartCalledOnce(t *testing.T) {
	tmp := t.TempDir()

	// Create an initialized project (go.mod exists)
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testproject\n\ngo 1.20\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create web directory structure like a real project
	cfg := app.NewConfig(tmp, func(...any) {})
	if err := os.MkdirAll(filepath.Join(tmp, cfg.WebDir()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, cfg.WebPublicDir()), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmp, cfg.WebPublicDir(), "index.html"), []byte("<html>Test</html>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Temporarily disable TestMode to allow AutoStart to be called
	originalTestMode := app.TestMode
	app.TestMode = false
	defer func() { app.TestMode = originalTestMode }()

	ctx := startTestApp(t, tmp)
	defer ctx.Cleanup()

	// Wait a bit more for all goroutines to settle (AutoStart has 100ms delay)
	time.Sleep(1 * time.Second)

	// Check AutoStart was called exactly once
	autoStartCalls := ctx.Browser.GetOpenCalls()

	if autoStartCalls != 1 {
		t.Errorf("BUG: Browser.AutoStart() was called %d times, expected exactly 1", autoStartCalls)
		t.Log("This indicates a duplicate browser open bug!")
		t.Logf("Logs:\n%s", ctx.Logs.String())
	} else {
		t.Logf("✅ Browser.AutoStart() called exactly once")
	}
}

// TestBrowserAutoStartNotCalledInWizard verifies that when starting the app
// in an empty directory (wizard mode), Browser.AutoStart() is NOT called
// until the wizard completes and project is created.
func TestBrowserAutoStartNotCalledInWizard(t *testing.T) {
	tmp := t.TempDir()

	// Empty directory - NO go.mod (wizard mode)

	ctx := startTestApp(t, tmp)
	defer ctx.Cleanup()

	// Wait a bit more for goroutines to settle
	time.Sleep(500 * time.Millisecond)

	// In wizard mode, AutoStart should NOT be called (project not ready yet)
	autoStartCalls := ctx.Browser.GetOpenCalls()

	if autoStartCalls != 0 {
		t.Errorf("In wizard mode (empty dir), Browser.OpenBrowser() was called %d times, expected 0", autoStartCalls)
		t.Log("Browser should only start after project is created")
	} else {
		t.Logf("✅ Browser.OpenBrowser() not called in wizard mode (correct)")
	}
}

// TestBrowserAutoStartInSubdirectory verifies that starting the app in a
// SUBDIRECTORY of an initialized project is now REJECTED.
func TestBrowserAutoStartInSubdirectory(t *testing.T) {
	tmp := t.TempDir()

	// 1. Create root project with go.mod
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testproject\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Create the subdirectory
	subDir := filepath.Join(tmp, "emptyfolder")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 3. Start app pointing to the SUBDIRECTORY - should return false/fail
	ExitChan := make(chan bool)
	logs := &SafeBuffer{}
	ui := newUiMockTest(logs.Log)
	db, _ := kvdb.New(filepath.Join(subDir, ".env"), logs.Log, app.NewMemoryStore())

	result := app.Start(subDir, logs.Log, ui, &MockBrowser{}, db, ExitChan, nil, nil, &MockGitClient{}, devflow.NewGoModHandler(), false, false, nil)

	if result != false {
		t.Errorf("Expected Start to return false for subdirectory execution, got true")
	}

	if !strings.Contains(logs.String(), "Directorio No Inicializado") && !strings.Contains(logs.String(), "Directory Not Initialized") {
		t.Errorf("Expected error message about uninitialized directory, got: %s", logs.String())
	}
}
