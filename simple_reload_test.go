package godev

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSimpleBrowserReload creates a single file, waits long enough for timer to expire
func TestSimpleBrowserReload(t *testing.T) {
	tmp := t.TempDir()

	var reloadCount int64

	logger := func(messages ...any) {
		var msg string
		for i, m := range messages {
			if i > 0 {
				msg += " "
			}
			msg += fmt.Sprint(m)
		}
		fmt.Printf("LOG: %s\n", msg)
	}

	// Start godev
	exitChan := make(chan bool)
	go Start(tmp, logger, exitChan)

	time.Sleep(200 * time.Millisecond)

	// Set up browser reload tracking
	SetWatcherBrowserReload(func() error {
		count := atomic.AddInt64(&reloadCount, 1)
		fmt.Printf("*** BROWSER RELOAD CALLED! Count: %d ***\n", count)
		return nil
	})

	time.Sleep(100 * time.Millisecond)

	// Create and modify ONE file, then wait a long time
	jsFile := filepath.Join(tmp, "modules", "test", "simple.js")
	require.NoError(t, os.MkdirAll(filepath.Dir(jsFile), 0755))
	require.NoError(t, os.WriteFile(jsFile, []byte("console.log('initial');"), 0644))

	fmt.Printf("=== File created, waiting for initial processing ===\n")
	time.Sleep(500 * time.Millisecond)

	initialCount := atomic.LoadInt64(&reloadCount)
	fmt.Printf("Reload count after initial creation: %d\n", initialCount)

	// Modify the file ONCE
	fmt.Printf("=== Single modification ===\n")
	require.NoError(t, os.WriteFile(jsFile, []byte("console.log('modified');"), 0644))

	// Wait long enough for timer to definitely expire (much longer than 100ms debounce)
	fmt.Printf("=== Waiting 1 second for timer to expire ===\n")
	time.Sleep(1 * time.Second)

	finalCount := atomic.LoadInt64(&reloadCount)
	fmt.Printf("Final reload count: %d\n", finalCount)

	exitChan <- true

	if finalCount > initialCount {
		t.Logf("✓ Browser reload was called %d times", finalCount-initialCount)
	} else {
		t.Errorf("Browser reload was never called even with single file modification and 1 second wait")
	}
}
