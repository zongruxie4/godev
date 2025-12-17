package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStartRealScenario reproduces the exact scenario from golite/test directory
// where multiple JS files exist but only the last one remains in main.js
func TestStartRealScenario(t *testing.T) {
	tmp := t.TempDir()

	// Create proper directory structure using Config methods (type-safe)
	goliteCfg := NewConfig(tmp, func(message ...any) {})

	// Create exact structure from real test directory
	files := map[string]string{
		"modules/users/newfile.js":       "console.log('H2');",
		"modules/medical/file1.js":       "console.log('one1');",
		"modules/medical/file2.js":       "console.log('two');",
		"modules/medical/file3.js":       "console.log(\"three\");",
		"modules/medical/file5.js":       "console.log('file5');",
		"modules/medical/mainconten1.js": "console.log('mainconten1');",
	}
	files[filepath.Join(goliteCfg.WebUIDir(), "theme.js")] = "console.log(\"Hello, PWA! 2\");"

	// Create directories and files BEFORE starting golite (like real scenario)
	for filePath, content := range files {
		fullPath := filepath.Join(tmp, filePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	// Capture logs
	var logs bytes.Buffer
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
		logs.WriteString(msg + "\n")
	}

	// Track browser reload calls
	var reloadCount int64
	reloadCalled := make(chan struct{}, 10) // Buffer for multiple reload events

	// Start golite like in real scenario
	exitChan := make(chan bool)
	go Start(tmp, logger, newUiMockTest(logger), exitChan)

	// Give a moment for Start to initialize and set ActiveHandler
	time.Sleep(50 * time.Millisecond)

	// Set up browser reload tracking after starting golite
	SetWatcherBrowserReload(func() error {
		atomic.AddInt64(&reloadCount, 1)
		select {
		case reloadCalled <- struct{}{}:
		default: // non-blocking in case buffer is full
		}
		return nil
	})

	// Give time to initialize and scan existing files
	time.Sleep(500 * time.Millisecond)

	// AssetMin generates script.js (not main.js) in the public directory
	scriptJsPath := filepath.Join(tmp, goliteCfg.WebPublicDir(), "script.js")

	// Check if script.js was created
	if _, err := os.Stat(scriptJsPath); os.IsNotExist(err) {
		// t.Logf("script.js not created yet, triggering a write event...")
		// Trigger a write event to make AssetMin write to disk
		testFilePath := filepath.Join(tmp, "modules", "medical", "file1.js")
		require.NoError(t, os.WriteFile(testFilePath, []byte("console.log('one1_modified');"), 0644))
		time.Sleep(200 * time.Millisecond)
		require.NoError(t, os.WriteFile(testFilePath, []byte("console.log('one1');"), 0644))
		time.Sleep(200 * time.Millisecond)
	}

	// Trigger additional JS file modifications to test browser reload
	// t.Logf("Triggering JS file modifications to test browser reload...")

	// Modify existing JS files to trigger reload events
	jsFiles := []string{
		filepath.Join(tmp, "modules", "users", "newfile.js"),
		filepath.Join(tmp, "modules", "medical", "file2.js"),
		filepath.Join(tmp, goliteCfg.WebUIDir(), "theme.js"),
	}

	for i, jsFile := range jsFiles {
		// t.Logf("Modifying %s (modification %d)", jsFile, i+1)
		content := fmt.Sprintf("console.log('modified_%d');", i+1)
		require.NoError(t, os.WriteFile(jsFile, []byte(content), 0644))
		time.Sleep(200 * time.Millisecond) // Wait longer than 150ms debounce timer
	}

	// Wait for final timer to expire
	time.Sleep(200 * time.Millisecond)

	// Read script.js content
	scriptJsContent, err := os.ReadFile(scriptJsPath)
	require.NoError(t, err, "script.js should exist")

	// Check what content should be present in script.js
	// Note: Files that were modified should contain their NEW content, not original
	expectedContents := []string{
		"modified_1",  // from users/newfile.js (was modified)
		"one1",        // from medical/file1.js (not modified)
		"modified_2",  // from medical/file2.js (was modified)
		"three",       // from medical/file3.js (not modified)
		"file5",       // from medical/file5.js (not modified)
		"mainconten1", // from medical/mainconten1.js (not modified)
		"modified_3",  // from web/ui/theme.js (was modified)
	}

	missing := []string{}
	for _, expected := range expectedContents {
		if !bytes.Contains(scriptJsContent, []byte(expected)) {
			missing = append(missing, expected)
		}
	}

	if len(missing) > 0 {
		t.Errorf("Missing content in script.js: %v", missing)
		t.Errorf("Expected content should reflect current state of files, not original content")
	} else {
		// t.Logf("✓ All expected content found in script.js (including modified files)")
	}

	// Verify browser reload was called during JS file modifications
	finalReloadCount := atomic.LoadInt64(&reloadCount)

	// We expect at least some reload calls since we modified JS files
	// The exact number may vary due to debouncing and initial registration
	if finalReloadCount == 0 {
		t.Errorf("Browser reload was never called, but JS files were modified")
	}

	// Stop the application
	exitChan <- true
}
