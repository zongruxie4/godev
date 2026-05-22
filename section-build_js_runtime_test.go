package app

import (
	"testing"
	"github.com/tinywasm/js"
	"github.com/tinywasm/client"
	"strings"
)

func TestSyncJSRuntime_GoMode(t *testing.T) {
	c := client.New(nil) // CurrentSizeMode = "L" (Go)

	syncJSRuntime(c)

	content := js.PageBootstrap().Content
	if !strings.Contains(content, "runtime.scheduleTimeoutEvent") {
		t.Errorf("Expected Go runtime signatures in bootstrap, got: %s", content)
	}
}

func TestSyncJSRuntime_TinyGoMode(t *testing.T) {
	c := client.New(nil)
	c.UpdateCurrentBuilder("M") // CurrentSizeMode = "M" (TinyGo)

	syncJSRuntime(c)

	content := js.PageBootstrap().Content
	if !strings.Contains(content, "sleepTicks") {
		t.Errorf("Expected TinyGo runtime signatures (sleepTicks) in bootstrap, got: %s", content)
	}
}

func TestSyncJSRuntime_ChangeOnHotReload(t *testing.T) {
	c := client.New(nil)

	// Start in Go mode
	syncJSRuntime(c)
	if !strings.Contains(js.PageBootstrap().Content, "runtime.scheduleTimeoutEvent") {
		t.Fatal("Initial Go mode failed")
	}

	// Switch to TinyGo
	c.UpdateCurrentBuilder("M")
	syncJSRuntime(c)
	if !strings.Contains(js.PageBootstrap().Content, "sleepTicks") {
		t.Errorf("Switch to TinyGo failed (missing sleepTicks), got: %s", js.PageBootstrap().Content)
	}
}

// TestSyncJSRuntime_ModeSwitch_BugReproduction reproduces the bug where switching to
// TinyGo mode (M or S) via UpdateCurrentBuilder does NOT produce a TinyGo script.js.
//
// Root cause: syncJSRuntime reads TinyGoCompilerFlag which is never set to true
// during a real mode switch — only CurrentSizeMode is updated by UpdateCurrentBuilder.
// Fix: syncJSRuntime must use RequiresTinyGo(CurrentSizeMode) instead of TinyGoCompilerFlag.
//
// Symptom in browser: "Import #0 wasi_snapshot_preview1: module is not an object or function"
// because the WASM binary compiled with TinyGo expects wasi_snapshot_preview1 but the
// generated script.js only provides the standard Go runtime (gojs imports).
func TestSyncJSRuntime_ModeSwitch_BugReproduction(t *testing.T) {
	c := client.New(nil) // starts at mode "L", TinyGoCompilerFlag = false

	// Simulate a mode switch to M (TinyGo) as Change() does — only CurrentSizeMode is updated.
	// TinyGoCompilerFlag is NOT set here, reproducing the real production flow.
	c.UpdateCurrentBuilder("M")

	// At this point: c.CurrentSizeMode == "M", c.TinyGoCompilerFlag == false (bug)
	if c.TinyGoCompilerFlag {
		t.Skip("TinyGoCompilerFlag is already true — bug may have been fixed in client; adjust test")
	}

	syncJSRuntime(c)

	content := js.PageBootstrap().Content

	// After switching to M (TinyGo), script.js MUST contain the TinyGo runtime.
	// Without the fix, this fails because syncJSRuntime uses TinyGoCompilerFlag (false)
	// and generates the Go runtime instead, missing wasi_snapshot_preview1.
	if !strings.Contains(content, "wasi_snapshot_preview1") && !strings.Contains(content, "sleepTicks") {
		t.Errorf("BUG: After UpdateCurrentBuilder(M), syncJSRuntime produced Go runtime instead of TinyGo runtime.\n"+
			"script.js will be missing wasi_snapshot_preview1, causing browser error:\n"+
			"  Import #0 wasi_snapshot_preview1: module is not an object or function\n"+
			"Fix: use c.RequiresTinyGo(c.CurrentSizeMode) in syncJSRuntime instead of c.TinyGoCompilerFlag")
	}
}
