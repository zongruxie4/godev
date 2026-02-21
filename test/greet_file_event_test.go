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

// TestGreetFileEventTriggersWasmCompilation simulates the exact user scenario:
// 1. app.Start tinywasm
// 2. Edit greet.go (dependency of main.go)
// 3. Verify WASM recompilation happens (not just Browser reload)
func TestGreetFileEventTriggersWasmCompilation(t *testing.T) {
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

	// Create greet.go (the dependency we'll edit)
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

	// Create web/client.go that imports greet (tinywasm's expected WASM entry point)
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

	// NOW run go mod tidy to populate go.sum (after files exist)
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

	// Track what happens
	var wasmCompilations int32
	var browserReloads int32

	tracker := func(messages ...any) {
		msg := strings.Join(func() []string {
			s := make([]string, len(messages))
			for i, m := range messages {
				s[i] = fmt.Sprint(m)
			}
			return s
		}(), " ")

		lowerMsg := strings.ToLower(msg)
		if (strings.Contains(msg, "WASM") && strings.Contains(lowerMsg, "compil")) || strings.Contains(msg, "WASM In-Memory") {
			atomic.AddInt32(&wasmCompilations, 1)
		}
		if strings.Contains(lowerMsg, "reload") {
			atomic.AddInt32(&browserReloads, 1)
		}
	}

	ctx := startTestApp(t, tmp, tracker)
	defer ctx.Cleanup()

	// Wait for initialization
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(ctx.Logs.String(), "Listening for File Changes") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Log("=== Initial state ready ===")
	initialCompilations := atomic.LoadInt32(&wasmCompilations)
	initialReloads := int32(ctx.Browser.GetReloadCalls())
	t.Logf("Initial compilations: %d, reloads: %d", initialCompilations, initialReloads)

	// Clear logs for cleaner output
	ctx.Logs.Clear()

	// Edit greet.go (simulate user editing the file)
	t.Log("\n=== Editing greet.go (dependency file) ===")
	updatedGreetContent := `package greet

import . "github.com/tinywasm/fmt"

func Greet(target string) string {
	return Sprintf("Hola, %s 👋", target, "from Go!!") // CHANGED: Hello -> Hola
}
`
	err = os.WriteFile(greetFile, []byte(updatedGreetContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for processing (100ms debounce + compilation + reload delay)
	compilationDeadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(compilationDeadline) {
		if atomic.LoadInt32(&wasmCompilations) > initialCompilations {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Check results
	finalCompilations := atomic.LoadInt32(&wasmCompilations)
	finalReloads := int32(ctx.Browser.GetReloadCalls())

	compilationsDelta := finalCompilations - initialCompilations
	reloadsDelta := finalReloads - initialReloads

	t.Log("\n=== Results ===")
	t.Logf("WASM compilations triggered: %d", compilationsDelta)
	t.Logf("Browser reloads triggered: %d", reloadsDelta)

	// Print relevant logs
	logOutput := ctx.Logs.String()
	t.Log("\n=== Captured Logs ===")
	for _, line := range strings.Split(logOutput, "\n") {
		lowerLine := strings.ToLower(line)
		if strings.Contains(line, "DEBUG") ||
			strings.Contains(line, "greet") ||
			strings.Contains(line, "WASM") ||
			strings.Contains(lowerLine, "compil") {
			t.Log(line)
		}
	}

	// Verify expectations
	if compilationsDelta == 0 {
		t.Error("❌ BUG CONFIRMED: Editing greet.go did NOT trigger WASM compilation!")
		t.Error("   Expected: greet.go change -> WASM recompile -> Browser reload")
		t.Error("   Actual: greet.go change -> Browser reload only (STALE WASM!)")
		t.Log("\nFull logs:")
		t.Log(logOutput)
	} else {
		t.Logf("✅ CORRECT: greet.go edit triggered %d WASM compilation(s)", compilationsDelta)
	}

	if reloadsDelta == 0 {
		t.Log("⚠️ No Browser reload happened (likely due to checking compilation failures in this test setup)")
	} else {
		t.Logf("✅ Browser reload happened (%d time(s))", reloadsDelta)
	}

	// Cleanup
	time.Sleep(200 * time.Millisecond)
}
