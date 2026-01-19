package test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tinywasm/app"
)

// TestSimpleBrowserReload creates a single file, waits long enough for timer to expire
func TestSimpleBrowserReload(t *testing.T) {
	tmp := t.TempDir()

	var mu sync.Mutex
	logger := func(messages ...any) {
		mu.Lock()
		defer mu.Unlock()
		var msg string
		for i, m := range messages {
			if i > 0 {
				msg += " "
			}
			msg += fmt.Sprint(m)
		}
		logIfVerbose(t, "LOG: %s", msg)
	}

	ExitChan := make(chan bool)
	// Set up Mock Browser injection
	mockBrowser := &MockBrowser{}
	app.SetInitialBrowser(mockBrowser)
	defer app.SetInitialBrowser(nil)

	// Create go.mod to pass the guard
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test"), 0644))

	// app.Start tinywasm
	go app.Start(tmp, logger, newUiMockTest(logger), ExitChan)
	// Wait for initialization
	h := app.WaitForActiveHandler(5 * time.Second)
	require.NotNil(t, h)

	time.Sleep(100 * time.Millisecond)

	// Create and modify ONE file, then wait a long time
	jsFile := filepath.Join(tmp, "modules", "test", "simple.js")
	require.NoError(t, os.MkdirAll(filepath.Dir(jsFile), 0755))
	require.NoError(t, os.WriteFile(jsFile, []byte("console.log('initial');"), 0644))

	logIfVerbose(t, "=== File created, waiting for initial processing ===")
	time.Sleep(500 * time.Millisecond)

	initialCount := mockBrowser.GetReloadCalls()
	logIfVerbose(t, "Reload count after initial creation: %d", initialCount)

	// Modify the file ONCE
	logIfVerbose(t, "=== Single modification ===")
	require.NoError(t, os.WriteFile(jsFile, []byte("console.log('modified');"), 0644))

	// Wait long enough for timer to definitely expire (much longer than 100ms debounce)
	logIfVerbose(t, "=== Waiting 1 second for timer to expire ===")
	time.Sleep(1 * time.Second)

	finalCount := mockBrowser.GetReloadCalls()
	logIfVerbose(t, "Final reload count: %d", finalCount)

	close(ExitChan)
	app.SetActiveHandler(nil)

	if finalCount > initialCount {
		// t.Logf("✓ Browser reload was called %d times", finalCount-initialCount)
	} else {
		t.Errorf("Browser reload was never called even with single file modification and 1 second wait")
	}
}
