package godev

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStartRealScenario reproduces the exact scenario from godev/test directory
// where multiple JS files exist but only the last one remains in main.js
func TestStartRealScenario(t *testing.T) {
	tmp := t.TempDir()

	// Create exact structure from real test directory
	files := map[string]string{
		"modules/users/newfile.js":       "console.log('H2');",
		"modules/medical/file1.js":       "console.log('one1');",
		"modules/medical/file2.js":       "console.log('two');",
		"modules/medical/file3.js":       "console.log(\"three\");",
		"modules/medical/file5.js":       "console.log('file5');",
		"modules/medical/mainconten1.js": "console.log('mainconten1');",
		"pwa/theme/main.js":              "console.log(\"Hello, PWA! 2\");",
	}

	// Create directories and files BEFORE starting godev (like real scenario)
	for filePath, content := range files {
		fullPath := filepath.Join(tmp, filePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	// Capture logs
	var logs bytes.Buffer
	logger := func(messages ...any) {
		var msg string
		for i, m := range messages {
			if i > 0 {
				msg += " "
			}
			msg += fmt.Sprint(m)
		}
		logs.WriteString(msg + "\n")
	}

	// Start godev like in real scenario
	exitChan := make(chan bool)
	go Start(tmp, logger, exitChan)

	// Give time to initialize and scan existing files
	time.Sleep(500 * time.Millisecond)

	mainJsPath := filepath.Join(tmp, "pwa", "public", "main.js")

	// Check if main.js was created
	if _, err := os.Stat(mainJsPath); os.IsNotExist(err) {
		t.Logf("main.js not created yet, triggering a write event...")
		// Trigger a write event to make AssetMin write to disk
		testFilePath := filepath.Join(tmp, "modules", "medical", "file1.js")
		require.NoError(t, os.WriteFile(testFilePath, []byte("console.log('one1_modified');"), 0644))
		time.Sleep(200 * time.Millisecond)
		require.NoError(t, os.WriteFile(testFilePath, []byte("console.log('one1');"), 0644))
		time.Sleep(200 * time.Millisecond)
	}

	// Read main.js content
	mainJsContent, err := os.ReadFile(mainJsPath)
	require.NoError(t, err, "main.js should exist")

	content := string(mainJsContent)
	t.Logf("main.js content: %s", content)
	t.Logf("Full logs:\n%s", logs.String())

	// Check what content is missing - these should all be present
	expectedContents := []string{
		"H2",            // from users/newfile.js
		"one1",          // from medical/file1.js
		"two",           // from medical/file2.js
		"three",         // from medical/file3.js
		"file5",         // from medical/file5.js
		"mainconten1",   // from medical/mainconten1.js
		"Hello, PWA! 2", // from pwa/theme/main.js
	}

	missing := []string{}
	for _, expected := range expectedContents {
		if !bytes.Contains(mainJsContent, []byte(expected)) {
			missing = append(missing, expected)
		}
	}

	if len(missing) > 0 {
		t.Errorf("Missing content in main.js: %v", missing)
		t.Errorf("This reproduces the real bug where only the last file is kept")
	}

	// Stop the application
	exitChan <- true
}
