package app

import (
	"testing"
	"github.com/tinywasm/js"
	"github.com/tinywasm/client"
	"strings"
)

func TestSyncJSRuntime_GoMode(t *testing.T) {
	c := &client.WasmClient{}
	c.TinyGoCompilerFlag = false

	syncJSRuntime(c)

	content := js.PageBootstrap().Content
	if !strings.Contains(content, "runtime.scheduleTimeoutEvent") {
		t.Errorf("Expected Go runtime signatures in bootstrap, got: %s", content)
	}
}

func TestSyncJSRuntime_TinyGoMode(t *testing.T) {
	c := &client.WasmClient{}
	c.TinyGoCompilerFlag = true

	syncJSRuntime(c)

	content := js.PageBootstrap().Content
	if !strings.Contains(content, "runtime.sleepTicks") {
		t.Errorf("Expected TinyGo runtime signatures (runtime.sleepTicks) in bootstrap, got: %s", content)
	}
}

func TestSyncJSRuntime_ChangeOnHotReload(t *testing.T) {
	c := &client.WasmClient{}

	// Start Go
	c.TinyGoCompilerFlag = false
	syncJSRuntime(c)
	if !strings.Contains(js.PageBootstrap().Content, "runtime.scheduleTimeoutEvent") {
		t.Fatal("Initial Go mode failed")
	}

	// Switch to TinyGo
	c.TinyGoCompilerFlag = true
	syncJSRuntime(c)
	if !strings.Contains(js.PageBootstrap().Content, "runtime.sleepTicks") {
		t.Errorf("Switch to TinyGo failed (missing runtime.sleepTicks), got: %s", js.PageBootstrap().Content)
	}
}
