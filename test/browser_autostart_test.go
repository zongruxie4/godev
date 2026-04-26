package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinywasm/app"
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
// direct subpackage (1 level under go.mod) is allowed — it must NOT produce
// a "Directory Not Initialized" rejection.
// Note: rejection of deep directories (2+ levels) is handled by main.go before
// Start() is called, so it is not tested here.
func TestBrowserAutoStartInSubdirectory(t *testing.T) {
	tmp := t.TempDir()

	// Create root project with go.mod
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testproject\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Direct subpackage (1 level) — must be allowed
	subDir := filepath.Join(tmp, "mycomponent")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	ctx := startTestApp(t, subDir)
	defer ctx.Cleanup()

	time.Sleep(300 * time.Millisecond)

	if strings.Contains(ctx.Logs.String(), "Directory Not Initialized") ||
		strings.Contains(ctx.Logs.String(), "Directorio No Inicializado") {
		t.Errorf("direct subpackage should be allowed, got rejection: %s", ctx.Logs.String())
	}
}
