package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tinywasm/app"
)

// TestBrowserAutoStartCalledOnce verifies that when starting the app in an
// initialized project directory, Browser.AutoStart() is called exactly once.
func TestBrowserAutoStartCalledOnce(t *testing.T) {
	tmp := t.TempDir()

	// Create an initialized project (go.mod exists)
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testproject\n\ngo 1.20\n"), 0644))

	// Create web directory structure like a real project
	cfg := app.NewConfig(tmp, func(...any) {})
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.WebDir()), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.WebPublicDir()), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmp, cfg.WebPublicDir(), "index.html"), []byte("<html>Test</html>"), 0644))

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

// TestBrowserAutoStartInSubdirectory verifies that when starting the app in a
// SUBDIRECTORY of an initialized project (e.g. 'emptyfolder' inside a project),
// Browser.AutoStart() is called exactly once.
func TestBrowserAutoStartInSubdirectory(t *testing.T) {
	tmp := t.TempDir()

	// 1. Create root project with go.mod
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testproject\n\ngo 1.25\n"), 0644))

	// 2. Create the subdirectory (reproduction of 'emptyfolder')
	subDir := filepath.Join(tmp, "emptyfolder")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// 3. Create web directory in root (because IsPartOfProject checks root)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "web"), 0755))

	// Temporarily disable TestMode to allow AutoStart
	originalTestMode := app.TestMode
	app.TestMode = false
	defer func() { app.TestMode = originalTestMode }()

	// 4. Start app pointing to the SUBDIRECTORY
	ctx := startTestApp(t, subDir)
	defer ctx.Cleanup()

	// Wait strictly enough for AutoStart
	time.Sleep(1000 * time.Millisecond) // generous time

	// Check calls
	autoStartCalls := ctx.Browser.GetOpenCalls()

	if autoStartCalls != 1 {
		t.Errorf("In Subdirectory, Browser.OpenBrowser() was called %d times, expected exactly 1", autoStartCalls)
		t.Logf("Logs:\n%s", ctx.Logs.String())
	} else {
		t.Logf("✅ In Subdirectory, Browser.OpenBrowser() called exactly once")
	}
}
