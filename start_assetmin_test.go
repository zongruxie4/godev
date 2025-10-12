package godev

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cdvelop/devtui"
	"github.com/stretchr/testify/require"
)

// TestStartAssetMinEventFlow initializes a handler via AddSectionBUILD and uses
// the internal AssetMin (h.assetsHandler) to send create/write events and
// verify main.js contains all expected modules. This mirrors assetmin/js_event_flow_test.go
func TestStartAssetMinEventFlow(t *testing.T) {
	tmp := t.TempDir()

	// prepare files
	file1Path := filepath.Join(tmp, "modules", "module1", "script1.js")
	file2Path := filepath.Join(tmp, "extras", "module2", "script2.js")
	file3Path := filepath.Join(tmp, "src", "webclient", "ui", "theme.js")

	require.NoError(t, os.MkdirAll(filepath.Dir(file1Path), 0755))
	require.NoError(t, os.MkdirAll(filepath.Dir(file2Path), 0755))
	require.NoError(t, os.MkdirAll(filepath.Dir(file3Path), 0755))

	file1Content := "console.log('Module One');"
	file2Content := "console.log('Module Two');"
	file3Content := "console.log('Theme Code');"

	require.NoError(t, os.WriteFile(file1Path, []byte(file1Content), 0644))
	require.NoError(t, os.WriteFile(file2Path, []byte(file2Content), 0644))
	require.NoError(t, os.WriteFile(file3Path, []byte(file3Content), 0644))

	// build a handler without starting goroutines
	h := &handler{rootDir: tmp, exitChan: make(chan bool)}

	// minimal tui so AddSectionBUILD can proceed
	h.tui = devtui.NewTUI(&devtui.TuiConfig{
		AppName:  "GODEV-TEST",
		ExitChan: h.exitChan,
		Color:    devtui.DefaultPalette(),
		Logger:   func(messages ...any) {},
	})

	// Initialize the build section which constructs the asset handler
	h.AddSectionBUILD()

	// Use the assetsHandler to send initial write events (simulate initial compilation)
	require.NoError(t, h.assetsHandler.NewFileEvent("script1.js", ".js", file1Path, "write"))
	require.NoError(t, h.assetsHandler.NewFileEvent("script2.js", ".js", file2Path, "write"))
	require.NoError(t, h.assetsHandler.NewFileEvent("theme.js", ".js", file3Path, "write"))

	mainJsPath := filepath.Join(tmp, "src", "webclient", "public", "main.js")

	// Wait for main.js to be created
	require.Eventually(t, func() bool {
		_, err := os.Stat(mainJsPath)
		return err == nil
	}, 3*time.Second, 50*time.Millisecond)

	initialMain, err := os.ReadFile(mainJsPath)
	require.NoError(t, err)

	// Send create events for same files (simulating watcher initial registration)
	require.NoError(t, h.assetsHandler.NewFileEvent("script1.js", ".js", file1Path, "create"))
	require.NoError(t, h.assetsHandler.NewFileEvent("script2.js", ".js", file2Path, "create"))
	require.NoError(t, h.assetsHandler.NewFileEvent("theme.js", ".js", file3Path, "create"))

	afterCreates, err := os.ReadFile(mainJsPath)
	require.NoError(t, err)
	require.True(t, bytes.Equal(initialMain, afterCreates), "main.js changed after duplicate 'create' events")

	// Create new empty file and create event
	newFilePath := filepath.Join(tmp, "modules", "module3", "newfile.js")
	require.NoError(t, os.MkdirAll(filepath.Dir(newFilePath), 0755))
	require.NoError(t, os.WriteFile(newFilePath, []byte{}, 0644))
	require.NoError(t, h.assetsHandler.NewFileEvent("newfile.js", ".js", newFilePath, "create"))

	afterEmptyCreate, err := os.ReadFile(mainJsPath)
	require.NoError(t, err)
	require.True(t, bytes.Equal(initialMain, afterEmptyCreate), "main.js changed after creating an empty file with 'create' event")

	// Write content and send write event
	addedContent := "console.log('New Module added');"
	require.NoError(t, os.WriteFile(newFilePath, []byte(addedContent), 0644))
	require.NoError(t, h.assetsHandler.NewFileEvent("newfile.js", ".js", newFilePath, "write"))

	finalMain, err := os.ReadFile(mainJsPath)
	require.NoError(t, err)
	finalStr := string(finalMain)

	require.Contains(t, finalStr, "Module One")
	require.Contains(t, finalStr, "Module Two")
	require.Contains(t, finalStr, "Theme Code")
	require.Contains(t, finalStr, "New Module added")
}
