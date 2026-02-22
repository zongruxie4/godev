package test

import (
	"path/filepath"
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/deploy"
	"github.com/tinywasm/wizard"
)

// TestDeploydTabAware verifies the Wizard and Deployd tab propagation logic
// effectively simulates what the App does during InitDeployHandlers
func TestDeploydTabAware(t *testing.T) {
	// Create mock Daemon using empty mockstore
	store := &mockStore{data: make(map[string]string)}
	d := deploy.NewDaemon(&deploy.DaemonConfig{
		AppRootDir:          "/tmp",
		CmdEdgeWorkerDir:    "cmd/worker",
		DeployEdgeWorkerDir: "public/worker",
		OutputWasmFileName:  "app.wasm",
		DeployConfigPath:    filepath.Join("/tmp", "deploy.yaml"),
		Store:               store,
	})

	// Setup a minimal wizard wrapping the Deployd similar to initDeployWizard
	cw := wizard.New(func(ctx *context.Context) {}, d)

	// Keep track of logs that reach the wizard's injected screen logger
	var logOutput string

	// mock TUI logger injected by DevTUI into the Wizard via AddHandler -> registerLoggableHandler
	tuiLogger := func(message ...any) {
		for _, m := range message {
			if str, ok := m.(string); ok {
				logOutput += str
			}
		}
	}
	// In DevTUI this happens internally inside AddHandler, we simulate the logger injection
	cw.SetLog(tuiLogger)

	// Simulate DevTUI changing to the Deploy tab.
	// This triggers OnTabActive on the wizard, and it should delegate to Deployd via OnWizardActive.
	cw.OnTabActive()
	// Simulate the DevTUI Start() double-fire issue
	cw.OnTabActive()

	// Verify that Deploy orchestrator successfully logged the menu string back through the wlog
	if len(logOutput) == 0 {
		t.Fatalf("Expected OnTabActive to trigger the deploy menu print, but no logs were found.")
	}

	count := countOccurrences(logOutput, "Available methods:")
	if count != 1 {
		t.Errorf("Expected deploy methods to be logged exactly 1 time, got: %d times. Log output: %s", count, logOutput)
	}

	t.Logf("Integration test works, log successfully proxied exactly once through Wizard -> Deployd -> Deploy: \n%s", logOutput)
}
