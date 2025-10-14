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

// TestStartJSEventFlow starts the application using Start(...) and reproduces the
// JS event flow used in assetmin/js_event_flow_test.go to check that main.js
// contains all files after create/write sequences when the watcher is active.
func TestStartJSEventFlow(t *testing.T) {
	// Setup temporary project layout
	tmp := t.TempDir()

	file1Path := filepath.Join(tmp, "modules", "module1", "script1.js")
	file2Path := filepath.Join(tmp, "extras", "module2", "script2.js")
	file3Path := filepath.Join(tmp, "src", "webclient", "ui", "theme.js")

	require.NoError(t, os.MkdirAll(filepath.Dir(file1Path), 0755))
	require.NoError(t, os.MkdirAll(filepath.Dir(file2Path), 0755))
	require.NoError(t, os.MkdirAll(filepath.Dir(file3Path), 0755))

	file1Content := "console.log('Module One');"
	file2Content := "console.log('Module Two');"
	file3Content := "console.log('Theme Code');"

	// Capture logs (simple writer)
	var logs bytes.Buffer
	logger := func(messages ...any) { // simple logger compatible with Start signature
		var msg string
		for i, m := range messages {
			if i > 0 {
				msg += " "
			}
			msg += fmt.Sprint(m)
		}
		logs.WriteString(msg + "\n")
	}

	// Start the application in a goroutine FIRST
	exitChan := make(chan bool)
	go Start(tmp, logger, newUiMockTest(logger), exitChan)

	// Give the services some time to initialize
	time.Sleep(250 * time.Millisecond)

	// Create files with initial content first
	require.NoError(t, os.WriteFile(file1Path, []byte(file1Content), 0644))
	require.NoError(t, os.WriteFile(file2Path, []byte(file2Content), 0644))
	require.NoError(t, os.WriteFile(file3Path, []byte(file3Content), 0644))

	// Wait a bit for CREATE events to be processed
	time.Sleep(200 * time.Millisecond)

	// Now modify files to trigger WRITE events that enable WriteOnDisk
	require.NoError(t, os.WriteFile(file1Path, []byte(file1Content+" // modified"), 0644))
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, os.WriteFile(file1Path, []byte(file1Content), 0644)) // restore content

	mainJsPath := filepath.Join(tmp, "src", "web", "public", "main.js")

	// Wait for main.js to be created after write events
	initialMain := waitForFile(t, mainJsPath, 3*time.Second)
	require.NotNil(t, initialMain, "main.js must exist after write events")

	// Create new empty JS file and then write content to it, relying on watcher to send events
	newFilePath := filepath.Join(tmp, "modules", "module3", "newfile.js")
	require.NoError(t, os.MkdirAll(filepath.Dir(newFilePath), 0755))
	require.NoError(t, os.WriteFile(newFilePath, []byte{}, 0644))

	// small pause for create event
	time.Sleep(100 * time.Millisecond)

	addedContent := "console.log('New Module added');"
	require.NoError(t, os.WriteFile(newFilePath, []byte(addedContent), 0644))

	// Wait for main.js to contain all expected strings
	expect := []string{"Module One", "Module Two", "Theme Code", "New Module added"}
	ok := waitForFileContains(t, mainJsPath, 5*time.Second, expect)
	require.True(t, ok, "final main.js should contain all expected modules; logs:\n%s", logs.String())

	// Stop the application
	exitChan <- true
}

// waitForFile polls until the file exists and returns its content or nil on timeout
func waitForFile(t *testing.T, path string, timeout time.Duration) []byte {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return data
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

// waitForFileContains polls until the file contains all substrings or times out
func waitForFileContains(t *testing.T, path string, timeout time.Duration, substrs []string) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			s := string(data)
			ok := true
			for _, sub := range substrs {
				if !bytes.Contains([]byte(s), []byte(sub)) {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
