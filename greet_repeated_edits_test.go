package app

import (
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

// TestGreetFileRepeatedEdits simulates the EXACT user scenario:
// Edit greet.go -> compiles -> Edit again -> should compile again
// This test reproduces the "sometimes works, sometimes doesn't" bug
func TestGreetFileRepeatedEdits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmp := t.TempDir()

	// Create config to get proper paths
	config := NewConfig(tmp, func(messages ...any) {})

	// Create realistic project structure (tinywasm expects web/ directory)
	err := os.MkdirAll(filepath.Join(tmp, config.WebDir()), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(tmp, "pkg/greet"), 0755)
	require.NoError(t, err)

	// Create go.mod
	goModContent := `module example

go 1.25.2

require github.com/tinywasm/fmt v0.12.3
`
	err = os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create greet.go
	greetFile := filepath.Join(tmp, "pkg/greet/greet.go")
	greetContent := `package greet

import . "github.com/tinywasm/fmt"

func Greet(target string) string {
	return Fmt("Hello, %s 👋", target, "from Go!!")
}
`
	err = os.WriteFile(greetFile, []byte(greetContent), 0644)
	require.NoError(t, err)
	// Create web/client.go (tinywasm's expected WASM entry point)
	// This file MUST exist before starting tinywasm, otherwise tinywasm creates a default one
	// without the greet import
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

	// Run go mod tidy
	// t.Log("Running go mod tidy...")
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmp
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Logf("go mod tidy output:\n%s", output)
		t.Fatalf("go mod tidy failed: %v", err)
	}

	// Track compilations
	var compilationCount int32
	logs := &SafeBuffer{}

	logger := func(messages ...any) {
		logs.Log(messages...)
		msg := logs.LastLog()

		if strings.Contains(msg, "Compiling WASM") {
			atomic.AddInt32(&compilationCount, 1)
			// t.Logf("[COMPILE %d] %s", count, msg)
		}
	}

	// Browser reload validation (stub)
	InitialBrowserReloadFunc = func() error { return nil }
	defer func() { InitialBrowserReloadFunc = nil }()

	exitChan := make(chan bool)
	go Start(tmp, logger, newUiMockTest(logger), exitChan)

	// Wait for initialization
	watcher := WaitWatcherReady(6 * time.Second)
	require.NotNil(t, watcher)
	h := GetActiveHandler()
	require.NotNil(t, h)

	// t.Log("\n=== TEST: Repeated edits to greet.go ===")

	// Perform 5 edits with realistic timing
	editMessages := []string{
		"Hello",
		"Hola",
		"Bonjour",
		"Ciao",
		"Hallo",
	}

	for i, msg := range editMessages {
		// t.Logf("\n--- Edit %d: Changing greeting to '%s' ---", i+1, msg)

		// Reset counter before edit
		beforeCount := atomic.LoadInt32(&compilationCount)

		// Edit the file
		newContent := fmt.Sprintf(`package greet

import . "github.com/tinywasm/fmt"

func Greet(target string) string {
	return Fmt("%s, %%s 👋", target, "from Go!!")
}
`, msg)

		err = os.WriteFile(greetFile, []byte(newContent), 0644)
		require.NoError(t, err)

		// Wait for compilation and reload
		time.Sleep(800 * time.Millisecond)

		afterCount := atomic.LoadInt32(&compilationCount)
		compiled := afterCount > beforeCount

		if compiled {
			// t.Logf("✅ Edit %d: Compilation triggered (total: %d)", i+1, afterCount)
		} else {
			t.Errorf("❌ Edit %d: NO compilation triggered! (total still: %d)", i+1, afterCount)
			t.Errorf("   Expected: Each edit should trigger compilation")
			t.Errorf("   Actual: Edit was ignored by the watcher")
		}

		// Small delay between edits (realistic user behavior)
		if i < len(editMessages)-1 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	// Cleanup
	exitChan <- true
	time.Sleep(200 * time.Millisecond)

	finalCount := atomic.LoadInt32(&compilationCount)
	// t.Logf("\n=== FINAL RESULTS ===")
	// t.Logf("Total edits: %d", len(editMessages))
	// t.Logf("Total compilations: %d", finalCount)

	if finalCount < int32(len(editMessages)) {
		t.Errorf("❌ BUG REPRODUCED: Only %d/%d edits triggered compilation!", finalCount, len(editMessages))
		t.Error("   This is the 'sometimes compiles, sometimes doesn't' bug!")
		t.Log("\n=== Full Logs ===")
		t.Log(logs.String())
	} else {
		// t.Logf("✅ SUCCESS: All %d edits triggered compilation consistently", len(editMessages))
	}
}
