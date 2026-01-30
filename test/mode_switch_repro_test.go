package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tinywasm/app"
)

// TestModeSwitchRepro reproduces the issue where switching compiler modes
// does not trigger a server restart, leading to mismatched WASM/JS.
func TestModeSwitchRepro(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test - skipping in short mode")
	}

	tmp := t.TempDir()

	// Initialize config and directories
	cfg := app.NewConfig(tmp, func(message ...any) {})
	appServerDir := filepath.Join(tmp, cfg.CmdAppServerDir())
	webPublicDir := filepath.Join(tmp, cfg.WebPublicDir())
	require.NoError(t, os.MkdirAll(appServerDir, 0755))
	require.NoError(t, os.MkdirAll(webPublicDir, 0755))

	// Create go.mod
	goModContent := `module testproject

go 1.20
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	// Create server.go (minimal)
	serverFilePath := filepath.Join(appServerDir, "server.go")
	serverContent := `package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
    if port == "" {
        port = "6060"
    }

	// Check for args
	args := os.Args[1:]
	fmt.Printf("Server args: %v\n", args)

	http.ListenAndServe(":"+port, nil)
}
`
	require.NoError(t, os.WriteFile(serverFilePath, []byte(serverContent), 0644))

	// Start App
	ctx := startTestApp(t, tmp)
	defer ctx.Cleanup()

	h := ctx.Handler

	// Enable External Mode
	require.NoError(t, h.ServerHandler.SetExternalServerMode(true))

	WaitForServerReady(t, h, 10*time.Second)

	// Switch Mode to Small (implying TinyGo)
	// This should ideally trigger a restart because WasmClient arguments change (ArgumentsForServer)
	// depending on implementation, we might need to verify that ArgumentsForServer actually changed first.
	// Currently relying on the fact that changing the mode SHOULD trigger OnWasmExecChange -> Restart.

	initialLogLen := ctx.Logs.Len()

	// We change the mode on the WasmClient handler
	// In the real app, this is done via TUI input, which calls Change()
	// h.WasmClient.Change("Size Mode:S")
	// But let's call the setter directly if exposed, or simulate Change
	h.WasmClient.Change("Size Mode:S")

	// Wait for potential restart
	time.Sleep(2 * time.Second)

	// Check logs for "Restarting External Server"
	logs := ctx.Logs.Lines()
	restarted := false
	for i := initialLogLen; i < len(logs); i++ {
		if logs[i] == "Restarting External Server..." || logs[i] == "External server restarted successfully" {
			restarted = true
			break
		}
	}

	// EXPECT FAILURE INITIALLY: The restart logic is not hooked up yet.
	if !restarted {
		t.Log("Reproduction: Server did NOT restart after mode switch as expected.")
		// We allow this to pass for now as we are reproducing the failure,
		// but in a real TDD cycle we would assert True here to fail the test.
		// For the purpose of the task, I will Assert it so we see it fail.
		t.Error("Server should have restarted after mode switch")
	}
}

func WaitForServerReady(t *testing.T, h *app.Handler, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Just a simple check, in reality we might check port
		if h.ServerHandler != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("Timeout waiting for server handler")
}
