package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tinywasm/app"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/kvdb"
)

// TestBrowserAutoStartCalledOnce verifies that when starting the app in an
// initialized project directory, Browser.AutoStart() is called exactly once.
// This reproduces the bug where the browser opens twice in some situations.
func TestBrowserAutoStartCalledOnce(t *testing.T) {
	tmp := t.TempDir()

	// Create an initialized project (go.mod exists)
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testproject\n\ngo 1.20\n"), 0644))

	// Create web directory structure like a real project
	cfg := app.NewConfig(tmp, func(messages ...any) {})
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.WebDir()), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.WebPublicDir()), 0755))

	// Create a basic index.html
	require.NoError(t, os.WriteFile(filepath.Join(tmp, cfg.WebPublicDir(), "index.html"), []byte("<html>Test</html>"), 0644))

	logs := &SafeBuffer{}
	logger := logs.Log

	// Inject MockBrowser to track AutoStart calls
	mockBrowser := &MockBrowser{}
	mockDB, _ := kvdb.New(filepath.Join(tmp, ".env"), logger, app.NewMemoryStore())

	// Temporarily disable TestMode to allow AutoStart to be called
	// (TestMode normally prevents browser opening in tests)
	originalTestMode := app.TestMode
	app.TestMode = false
	defer func() { app.TestMode = originalTestMode }()

	ExitChan := make(chan bool)
	go app.Start(tmp, logger, newUiMockTest(logger), mockBrowser, mockDB, ExitChan, devflow.NewMockGitHubAuth())

	// Wait for initialization
	h := app.WaitForActiveHandler(5 * time.Second)
	require.NotNil(t, h, "Handler should be initialized")

	// Wait a bit more for all goroutines to settle (AutoStart has 100ms delay)
	time.Sleep(500 * time.Millisecond)

	// Check AutoStart was called exactly once
	autoStartCalls := mockBrowser.GetOpenCalls()

	// Cleanup
	close(ExitChan)
	app.SetActiveHandler(nil)
	time.Sleep(100 * time.Millisecond)

	if autoStartCalls != 1 {
		t.Errorf("BUG: Browser.AutoStart() was called %d times, expected exactly 1", autoStartCalls)
		t.Log("This indicates a duplicate browser open bug!")
		t.Logf("Logs:\n%s", logs.String())
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

	logs := &SafeBuffer{}
	logger := logs.Log

	// Inject MockBrowser to track AutoStart calls
	mockBrowser := &MockBrowser{}

	ExitChan := make(chan bool)
	mockDB, _ := kvdb.New(filepath.Join(tmp, ".env"), logger, app.NewMemoryStore())
	go app.Start(tmp, logger, newUiMockTest(logger), mockBrowser, mockDB, ExitChan, devflow.NewMockGitHubAuth())

	// Wait a bit for initialization
	h := app.WaitForActiveHandler(5 * time.Second)
	require.NotNil(t, h, "Handler should be initialized")

	// Wait a bit more for goroutines to settle
	time.Sleep(500 * time.Millisecond)

	// In wizard mode, AutoStart should NOT be called (project not ready yet)
	autoStartCalls := mockBrowser.GetOpenCalls()

	// Cleanup
	close(ExitChan)
	app.SetActiveHandler(nil)
	time.Sleep(100 * time.Millisecond)

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
//
// This specifically reproduces the scenario where the user runs the app from
// a folder inside the project, which IsPartOfProject() detects as valid.
func TestBrowserAutoStartInSubdirectory(t *testing.T) {
	tmp := t.TempDir()

	// 1. Create root project with go.mod
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module testproject\n\ngo 1.25\n"), 0644))

	// 2. Create the subdirectory (reproduction of 'emptyfolder')
	subDir := filepath.Join(tmp, "emptyfolder")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// 3. Create web directory in root (because IsPartOfProject checks root)
	// But wait, IsPartOfProject looks up. If logic depends on web dir presence, we add it.
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "web"), 0755))

	logs := &SafeBuffer{}
	logger := logs.Log

	// Inject MockBrowser
	mockBrowser := &MockBrowser{}

	// Temporarily disable TestMode to allow AutoStart
	originalTestMode := app.TestMode
	app.TestMode = false
	defer func() { app.TestMode = originalTestMode }()

	ExitChan := make(chan bool)

	// 4. Start app pointing to the SUBDIRECTORY
	mockDB, _ := kvdb.New(filepath.Join(tmp, ".env"), logger, app.NewMemoryStore())
	go app.Start(subDir, logger, newUiMockTest(logger), mockBrowser, mockDB, ExitChan, devflow.NewMockGitHubAuth())

	// Wait for initialization
	h := app.WaitForActiveHandler(5 * time.Second)
	require.NotNil(t, h, "Handler should be initialized")

	// Wait strictly enough for AutoStart
	time.Sleep(1000 * time.Millisecond) // generous time

	// Check calls
	autoStartCalls := mockBrowser.GetOpenCalls()
	// Note: since IsPartOfProject returns true (due to parent go.mod),
	// logic skips Wizard and calls OnProjectReady directly.
	// We expect exactly 1 OpenBrowser call.

	// Cleanup
	close(ExitChan)
	app.SetActiveHandler(nil)
	time.Sleep(100 * time.Millisecond)

	if autoStartCalls != 1 {
		t.Errorf("In Subdirectory, Browser.OpenBrowser() was called %d times, expected exactly 1", autoStartCalls)
		t.Logf("Logs:\n%s", logs.String())
	} else {
		t.Logf("✅ In Subdirectory, Browser.OpenBrowser() called exactly once")
	}
}
