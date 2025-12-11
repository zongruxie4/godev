package tinywasm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGreetFileEventTriggersWasmCompilation simulates the exact user scenario:
// 1. Start golite
// 2. Edit greet.go (dependency of main.go)
// 3. Verify WASM recompilation happens (not just browser reload)
func TestGreetFileEventTriggersWasmCompilation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmp := t.TempDir()

	// Create config to get proper paths
	config := NewConfig(tmp, func(messages ...any) {})

	// Create realistic project structure (golite expects web/ directory)
	err := os.MkdirAll(filepath.Join(tmp, config.WebDir()), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(tmp, "pkg/greet"), 0755)
	require.NoError(t, err)

	// Create go.mod
	goModContent := `module example

go 1.25.2

require github.com/tinywasm/fmt v0.8.3
`
	err = os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create greet.go (the dependency we'll edit)
	greetFile := filepath.Join(tmp, "pkg/greet/greet.go")
	greetContent := `package greet

import . "github.com/tinywasm/fmt"

func Greet(target string) string {
	return Fmt("Hello, %s 👋", target, "from Go!!")
}
`
	err = os.WriteFile(greetFile, []byte(greetContent), 0644)
	require.NoError(t, err)

	// Create web/client.go that imports greet (golite's expected WASM entry point)
	// This file MUST exist before starting golite with the greet import
	clientGoFile := filepath.Join(tmp, config.WebDir(), config.ClientFileName())
	clientGoContent := `//go:build wasm

package main

import (
	"example/pkg/greet"
	"syscall/js"
)

func main() {
	dom := js.Global().Get("document").Call("createElement", "div")
	dom.Set("innerHTML", greet.Greet("WebAssembly!"))
	body := js.Global().Get("document").Get("body")
	body.Call("appendChild", dom)
	select {}
}
`
	err = os.WriteFile(clientGoFile, []byte(clientGoContent), 0644)
	require.NoError(t, err)

	// NOW run go mod tidy to populate go.sum (after files exist)
	t.Log("Running go mod tidy to populate dependencies...")
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmp
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Logf("go mod tidy output:\n%s", output)
		t.Fatalf("go mod tidy failed: %v", err)
	}

	// Verify go.sum was created
	if _, err := os.Stat(filepath.Join(tmp, "go.sum")); err != nil {
		t.Fatalf("go.sum not created: %v", err)
	}
	t.Log("✓ Dependencies ready")

	// Track what happens
	var wasmCompilations int32
	var browserReloads int32
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

		// Track compilations and reloads
		if strings.Contains(msg, "WASM") && strings.Contains(msg, "compil") {
			atomic.AddInt32(&wasmCompilations, 1)
		}
		if strings.Contains(msg, "reload") || strings.Contains(msg, "Reload") {
			atomic.AddInt32(&browserReloads, 1)
		}
	}

	exitChan := make(chan bool)

	// Start golite
	go Start(tmp, logger, newUiMockTest(logger), exitChan)

	// Wait for initialization
	time.Sleep(500 * time.Millisecond)

	require.NotNil(t, ActiveHandler, "ActiveHandler should be set")
	require.NotNil(t, ActiveHandler.watcher, "Watcher should be initialized")

	// Spy on browser reload calls
	reloadChan := make(chan struct{}, 10)
	SetWatcherBrowserReload(func() error {
		atomic.AddInt32(&browserReloads, 1)
		select {
		case reloadChan <- struct{}{}:
		default:
		}
		return nil
	})

	t.Log("=== Initial state ready ===")
	initialCompilations := atomic.LoadInt32(&wasmCompilations)
	initialReloads := atomic.LoadInt32(&browserReloads)
	t.Logf("Initial compilations: %d, reloads: %d", initialCompilations, initialReloads)

	// Clear logs for cleaner output
	logs.Reset()

	// Edit greet.go (simulate user editing the file)
	t.Log("\n=== Editing greet.go (dependency file) ===")
	updatedGreetContent := `package greet

import . "github.com/tinywasm/fmt"

func Greet(target string) string {
	return Fmt("Hola, %s 👋", target, "from Go!!") // CHANGED: Hello -> Hola
}
`
	err = os.WriteFile(greetFile, []byte(updatedGreetContent), 0644)
	require.NoError(t, err)

	// Wait for processing (100ms debounce + compilation + reload delay)
	time.Sleep(1 * time.Second)

	// Check results
	finalCompilations := atomic.LoadInt32(&wasmCompilations)
	finalReloads := atomic.LoadInt32(&browserReloads)

	compilationsDelta := finalCompilations - initialCompilations
	reloadsDelta := finalReloads - initialReloads

	t.Log("\n=== Results ===")
	t.Logf("WASM compilations triggered: %d", compilationsDelta)
	t.Logf("Browser reloads triggered: %d", reloadsDelta)

	// Print relevant logs
	logOutput := logs.String()
	t.Log("\n=== Captured Logs ===")
	for _, line := range strings.Split(logOutput, "\n") {
		if strings.Contains(line, "DEBUG") ||
			strings.Contains(line, "greet") ||
			strings.Contains(line, "WASM") ||
			strings.Contains(line, "ThisFileIsMine") {
			t.Log(line)
		}
	}

	// Verify expectations
	if compilationsDelta == 0 {
		t.Error("❌ BUG CONFIRMED: Editing greet.go did NOT trigger WASM compilation!")
		t.Error("   Expected: greet.go change -> WASM recompile -> browser reload")
		t.Error("   Actual: greet.go change -> browser reload only (STALE WASM!)")
		t.Log("\nFull logs:")
		t.Log(logOutput)
	} else {
		t.Logf("✅ CORRECT: greet.go edit triggered %d WASM compilation(s)", compilationsDelta)
	}

	if reloadsDelta == 0 {
		t.Error("❌ No browser reload happened")
	} else {
		t.Logf("✅ Browser reload happened (%d time(s))", reloadsDelta)
	}

	// Cleanup
	exitChan <- true
	time.Sleep(200 * time.Millisecond)
}
