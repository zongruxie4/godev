package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tinywasm/app"
)

// TestGreetFileRepeatedEdits simulates the EXACT user scenario:
// Edit greet.go -> compiles -> Edit again -> should compile again
func TestGreetFileRepeatedEdits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmp := t.TempDir()

	// Create Config to get proper paths
	Config := app.NewConfig(tmp, func(messages ...any) {})

	// Create realistic project structure (tinywasm expects web/ directory)
	err := os.MkdirAll(filepath.Join(tmp, Config.WebDir()), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(filepath.Join(tmp, "pkg/greet"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goModContent := `module example

go 1.25.2

require github.com/tinywasm/fmt v0.17.1
`
	err = os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create greet.go
	greetFile := filepath.Join(tmp, "pkg/greet/greet.go")
	greetContent := `package greet

import . "github.com/tinywasm/fmt"

func Greet(target string) string {
	return Sprintf("Hello, %s 👋", target, "from Go!!")
}
`
	err = os.WriteFile(greetFile, []byte(greetContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	// Create web/client.go (tinywasm's expected WASM entry point)
	clientGoFile := filepath.Join(tmp, Config.WebDir(), Config.ClientFileName())
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
	if err != nil {
		t.Fatal(err)
	}

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmp
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Logf("go mod tidy output:\n%s", output)
		t.Fatalf("go mod tidy failed: %v", err)
	}

	// Track compilations
	var compilationCount int32
	tracker := func(messages ...any) {
		msg := strings.Join(func() []string {
			s := make([]string, len(messages))
			for i, m := range messages {
				s[i] = fmt.Sprint(m)
			}
			return s
		}(), " ")

		if strings.Contains(msg, "Compiling WASM") || strings.Contains(msg, "WASM In-Memory") {
			atomic.AddInt32(&compilationCount, 1)
		}
	}

	ctx := startTestApp(t, tmp, tracker)
	defer ctx.Cleanup()

	// Wait for initialization
	Watcher := app.WaitWatcherReady(6 * time.Second)
	if Watcher == nil {
		t.Fatal("Watcher is nil")
	}

	// Perform 5 edits with realistic timing
	editMessages := []string{
		"Hello",
		"Hola",
		"Bonjour",
		"Ciao",
		"Hallo",
	}

	for i, msg := range editMessages {
		// Reset counter before edit
		beforeCount := atomic.LoadInt32(&compilationCount)

		// Small delay BEFORE edit to ensure stability (events from init or previous edits settled)
		time.Sleep(100 * time.Millisecond)

		// Edit the file
		newContent := fmt.Sprintf(`package greet

import . "github.com/tinywasm/fmt"

func Greet(target string) string {
	return Sprintf("%s, %%s 👋", target, "from Go!!")
}
`, msg)

		err = os.WriteFile(greetFile, []byte(newContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Wait for compilation and reload (poll instead of fixed sleep)
		// Increased deadline to 2s to handle slow CI/parallel runs
		deadline := time.Now().Add(2000 * time.Millisecond)
		var compiled bool
		for time.Now().Before(deadline) {
			if atomic.LoadInt32(&compilationCount) > beforeCount {
				compiled = true
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		if !compiled {
			t.Errorf("❌ Edit %d (%s): NO compilation triggered! (current total: %d)", i+1, msg, atomic.LoadInt32(&compilationCount))
			t.Errorf("   Expected: Each edit should trigger exactly ONE compilation")
			t.Errorf("   Log snippet:\n%s", ctx.Logs.String())
		}
	}

	finalCount := atomic.LoadInt32(&compilationCount)

	if finalCount < int32(len(editMessages)) {
		t.Errorf("❌ BUG REPRODUCED: Only %d/%d edits triggered compilation!", finalCount, len(editMessages))
		t.Error("   This is the 'sometimes compiles, sometimes doesn't' bug!")
		t.Log("\n=== Full Logs ===")
		t.Log(ctx.Logs.String())
	}
}
