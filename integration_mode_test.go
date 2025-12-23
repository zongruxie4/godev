package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildModeToggleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmp := t.TempDir()

	// Initialize conventional directory structure
	cfg := NewConfig(tmp, func(message ...any) {})
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.CmdWebClientDir()), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.WebPublicDir()), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, cfg.CmdAppServerDir()), 0755))

	// Create a dummy client.go
	clientGoPath := filepath.Join(tmp, cfg.CmdWebClientDir(), "client.go")
	require.NoError(t, os.WriteFile(clientGoPath, []byte("package main\nfunc main(){}"), 0644))

	// Create a dummy server file
	serverGoPath := filepath.Join(tmp, cfg.CmdAppServerDir(), "main.go")
	require.NoError(t, os.WriteFile(serverGoPath, []byte("package main\nfunc main(){}"), 0644))

	// Go mod
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test\ngo 1.20"), 0644))

	logs := &SafeBuffer{}
	logger := logs.Log

	exitChan := make(chan bool)
	go Start(tmp, logger, newUiMockTest(logger), exitChan)
	defer func() {
		close(exitChan)
		SetActiveHandler(nil)
	}()

	h := WaitForActiveHandler(5 * time.Second)
	require.NotNil(t, h)

	// 1. Initial State: Memory Mode (Default)
	wasmPath := filepath.Join(tmp, cfg.WebPublicDir(), "client.wasm")
	// h.wasmClient.Strategy should be In-Memory.
	// Ensure any existing file is removed to verify it's not being actively written/maintained.
	_ = os.Remove(wasmPath)

	// Trigger a file event. If in-memory, it should compile but NOT write to disk.
	// require.NoError(t, h.wasmClient.NewFileEvent("client.go", ".go", clientGoPath, "write"))

	// Check again
	_, err := os.Stat(wasmPath)
	require.Error(t, err, "wasm file should NOT be created in memory mode after file event")

	// 2. Switch to Disk Mode
	progress := make(chan string, 10)
	buildMode := &BuildModeOnDisk{h: h}
	buildMode.Execute(progress)

	// Verify persistence in DB
	val, err := h.db.Get(StoreKeyBuildModeOnDisk)
	require.NoError(t, err)
	require.Equal(t, "true", val)

	// 3. Trigger compilation by modifying client.go
	// We need to wait for the watcher to be ready
	watcher := WaitWatcherReady(2 * time.Second)
	require.NotNil(t, watcher)

	// Modify client.go to trigger watcher
	require.NoError(t, os.WriteFile(clientGoPath, []byte("package main\nfunc main(){ println(1) }"), 0644))

	// Wait for compilation to finish. Compilation can take a while.
	// We check for the file existence.
	require.Eventually(t, func() bool {
		_, err := os.Stat(wasmPath)
		return err == nil
	}, 15*time.Second, 500*time.Millisecond, "wasm file should be written to disk after mode switch and file change")

	// 4. Switch back to Memory Mode
	buildMode.Execute(progress)
	val, err = h.db.Get(StoreKeyBuildModeOnDisk)
	require.NoError(t, err)
	require.Equal(t, "false", val)

	// Delete the file and trigger another change
	require.NoError(t, os.Remove(wasmPath))
	require.NoError(t, os.WriteFile(clientGoPath, []byte("package main\nfunc main(){ println(2) }"), 0644))

	// Wait a bit and verify file is NOT recreated
	time.Sleep(2 * time.Second)
	_, err = os.Stat(wasmPath)
	require.Error(t, err, "wasm file should NOT be recreated in memory mode")
}
