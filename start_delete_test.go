package golite

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStartDeleteFileScenario tests that when a file is deleted, its content
// is removed from main.js output
func TestStartDeleteFileScenario(t *testing.T) {
	tmp := t.TempDir()

	// Create proper directory structure using Config methods (type-safe)
	goliteCfg := NewConfig(tmp, func(message ...any) {})

	// Create initial files
	files := map[string]string{
		"modules/file1.js": "console.log('file1');",
		"modules/file2.js": "console.log('file2');",
		"modules/file3.js": "console.log('file3');",
	}

	// Create directories and files
	for filePath, content := range files {
		fullPath := filepath.Join(tmp, filePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	// Start golite
	exitChan := make(chan bool)
	go Start(tmp, nil, newUiMockTest(nil), exitChan)

	// Give time to initialize and process initial files
	time.Sleep(500 * time.Millisecond)

	// AssetMin generates script.js (not main.js) in the public directory
	scriptJsPath := filepath.Join(tmp, goliteCfg.WebPublicDir(), "script.js")

	// Trigger initial write to create main.js
	file1Path := filepath.Join(tmp, "modules", "file1.js")
	require.NoError(t, os.WriteFile(file1Path, []byte("console.log('file1_modified');"), 0644))
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, os.WriteFile(file1Path, []byte("console.log('file1');"), 0644))
	time.Sleep(200 * time.Millisecond)

	// Read initial script.js content
	initialContent, err := os.ReadFile(scriptJsPath)
	require.NoError(t, err, "script.js should exist")

	// Verify all files are present initially
	require.Contains(t, string(initialContent), "file1", "file1 should be in script.js")
	require.Contains(t, string(initialContent), "file2", "file2 should be in script.js")
	require.Contains(t, string(initialContent), "file3", "file3 should be in script.js")

	// Now DELETE file2
	file2Path := filepath.Join(tmp, "modules", "file2.js")
	require.NoError(t, os.Remove(file2Path))

	// Wait for delete event to be processed
	time.Sleep(500 * time.Millisecond)

	// Read script.js content after deletion
	afterDeleteContent, err := os.ReadFile(scriptJsPath)
	require.NoError(t, err, "script.js should still exist")

	// Verify file2 content is removed but file1 and file3 remain
	require.Contains(t, string(afterDeleteContent), "file1", "file1 should still be in script.js")
	require.NotContains(t, string(afterDeleteContent), "file2", "file2 should be REMOVED from script.js")
	require.Contains(t, string(afterDeleteContent), "file3", "file3 should still be in script.js")

	// Stop the application
	exitChan <- true
}
