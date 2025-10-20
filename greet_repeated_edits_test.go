package golite

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

// TestGreetFileRepeatedEdits simulates the EXACT user scenario:
// Edit greet.go -> compiles -> Edit again -> should compile again
// This test reproduces the "sometimes works, sometimes doesn't" bug
func TestGreetFileRepeatedEdits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmp := t.TempDir()

	// Create realistic project structure
	err := os.MkdirAll(filepath.Join(tmp, "src/cmd/webclient"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(tmp, "src/pkg/greet"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(tmp, "src/web/public"), 0755)
	require.NoError(t, err)

	// Create go.mod
	goModContent := `module example

go 1.25.2

require github.com/cdvelop/tinystring v0.8.3
`
	err = os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Create greet.go
	greetFile := filepath.Join(tmp, "src/pkg/greet/greet.go")
	greetContent := `package greet

import . "github.com/cdvelop/tinystring"

func Greet(target string) string {
	return Fmt("Hello, %s 👋", target, "from Go!!")
}
`
	err = os.WriteFile(greetFile, []byte(greetContent), 0644)
	require.NoError(t, err)

	// Create main.go
	mainGoFile := filepath.Join(tmp, "src/cmd/webclient/main.go")
	mainGoContent := `package main

import (
	"example/src/pkg/greet"
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
	err = os.WriteFile(mainGoFile, []byte(mainGoContent), 0644)
	require.NoError(t, err)

	// Run go mod tidy
	t.Log("Running go mod tidy...")
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmp
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Logf("go mod tidy output:\n%s", output)
		t.Fatalf("go mod tidy failed: %v", err)
	}

	// Track compilations
	var compilationCount int32
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

		if strings.Contains(msg, "Compiling WASM") {
			count := atomic.AddInt32(&compilationCount, 1)
			t.Logf("[COMPILE %d] %s", count, msg)
		}
	}

	exitChan := make(chan bool)
	go Start(tmp, logger, newUiMockTest(logger), exitChan)
	time.Sleep(500 * time.Millisecond)

	require.NotNil(t, ActiveHandler)
	require.NotNil(t, ActiveHandler.watcher)

	t.Log("\n=== TEST: Repeated edits to greet.go ===")

	// Perform 5 edits with realistic timing
	editMessages := []string{
		"Hello",
		"Hola",
		"Bonjour",
		"Ciao",
		"Hallo",
	}

	for i, msg := range editMessages {
		t.Logf("\n--- Edit %d: Changing greeting to '%s' ---", i+1, msg)

		// Reset counter before edit
		beforeCount := atomic.LoadInt32(&compilationCount)

		// Edit the file
		newContent := fmt.Sprintf(`package greet

import . "github.com/cdvelop/tinystring"

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
			t.Logf("✅ Edit %d: Compilation triggered (total: %d)", i+1, afterCount)
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
	t.Logf("\n=== FINAL RESULTS ===")
	t.Logf("Total edits: %d", len(editMessages))
	t.Logf("Total compilations: %d", finalCount)

	if finalCount < int32(len(editMessages)) {
		t.Errorf("❌ BUG REPRODUCED: Only %d/%d edits triggered compilation!", finalCount, len(editMessages))
		t.Error("   This is the 'sometimes compiles, sometimes doesn't' bug!")
		t.Log("\n=== Full Logs ===")
		t.Log(logs.String())
	} else {
		t.Logf("✅ SUCCESS: All %d edits triggered compilation consistently", len(editMessages))
	}
}
