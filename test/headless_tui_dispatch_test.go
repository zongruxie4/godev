package test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/tinywasm/app"
)

// mockHandler implements all necessary interfaces
type mockHandler struct {
	name         string
	value        string
	label        string
	changeVal    string
	executeCount int
	mu           sync.Mutex
	shortcuts    []map[string]string
}

func (m *mockHandler) Name() string { return m.name }
func (m *mockHandler) Value() string { return m.value }
func (m *mockHandler) Label() string { return m.label }
func (m *mockHandler) Execute() { m.mu.Lock(); m.executeCount++; m.mu.Unlock() }
func (m *mockHandler) Change(v string) { m.mu.Lock(); m.changeVal = v; m.mu.Unlock() }
func (m *mockHandler) Shortcuts() []map[string]string { return m.shortcuts }

// TestHeadlessTUI_EditHandler_DispatchByName verifies edit handlers dispatch by handler name
func TestHeadlessTUI_EditHandler_DispatchByName(t *testing.T) {
	tui := app.NewHeadlessTUI(func(msg ...any) {})

	handler := &mockHandler{name: "TestEdit", value: "initial", label: "Test Label"}
	section := &headlessSection{Title: "TEST"}

	tui.AddHandler(handler, "#FF0000", section)

	// Dispatch by handler name should call Change
	dispatched := tui.DispatchAction("TestEdit", "newvalue")
	if !dispatched {
		t.Fatal("DispatchAction should have found handler by name")
	}

	// DispatchAction runs in goroutine, give it time
	waitForChange := make(chan bool, 1)
	go func() {
		for i := 0; i < 100; i++ {
			handler.mu.Lock()
			val := handler.changeVal
			handler.mu.Unlock()
			if val == "newvalue" {
				waitForChange <- true
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		waitForChange <- false
	}()

	select {
	case success := <-waitForChange:
		if !success {
			t.Error("changeVal was never set to 'newvalue'")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Change to be called")
	}
}

// TestHeadlessTUI_ShortcutProvider_DispatchByKey verifies shortcut keys dispatch to handler
func TestHeadlessTUI_ShortcutProvider_DispatchByKey(t *testing.T) {
	tui := app.NewHeadlessTUI(func(msg ...any) {})

	handler := &mockHandler{
		name:      "WASM",
		value:     "M",
		shortcuts: []map[string]string{
			{"L": "Large"},
			{"M": "Medium"},
			{"S": "Small"},
		},
	}
	section := &headlessSection{Title: "BUILD"}

	tui.AddHandler(handler, "#00DD00", section)

	// Dispatch by shortcut key "L"
	dispatched := tui.DispatchAction("L", "L")
	if !dispatched {
		t.Fatal("DispatchAction should have found shortcut 'L'")
	}

	// Wait for goroutine to execute
	waitForChange := make(chan bool, 1)
	go func() {
		for i := 0; i < 100; i++ {
			handler.mu.Lock()
			val := handler.changeVal
			handler.mu.Unlock()
			if val == "L" {
				waitForChange <- true
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		waitForChange <- false
	}()

	select {
	case success := <-waitForChange:
		if !success {
			t.Error("changeVal was never set to 'L'")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Change to be called")
	}
}

// TestHeadlessTUI_GetHandlerStates_IncludesShortcuts verifies state includes shortcuts
func TestHeadlessTUI_GetHandlerStates_IncludesShortcuts(t *testing.T) {
	tui := app.NewHeadlessTUI(func(msg ...any) {})

	handler := &mockHandler{
		name:  "WasmCompiler",
		value: "M",
		label: "Size",
		shortcuts: []map[string]string{
			{"L": "Large"},
			{"M": "Medium"},
		},
	}
	section := &headlessSection{Title: "BUILD"}

	tui.AddHandler(handler, "#0000FF", section)

	data := tui.GetHandlerStates()

	var entries []struct {
		HandlerName string              `json:"handler_name"`
		Shortcut    string              `json:"shortcut"`
		Shortcuts   []map[string]string `json:"shortcuts"`
	}

	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("failed to unmarshal state: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one entry")
	}

	entry := entries[0]
	if entry.HandlerName != "WasmCompiler" {
		t.Errorf("expected HandlerName='WasmCompiler', got %q", entry.HandlerName)
	}

	if entry.Shortcut != "WasmCompiler" {
		t.Errorf("expected Shortcut='WasmCompiler' (handler name), got %q", entry.Shortcut)
	}

	if len(entry.Shortcuts) != 2 {
		t.Errorf("expected 2 shortcuts, got %d", len(entry.Shortcuts))
	}
}

// TestHeadlessTUI_ExecutionHandler_DispatchByName verifies execution handlers dispatch by name
func TestHeadlessTUI_ExecutionHandler_DispatchByName(t *testing.T) {
	tui := app.NewHeadlessTUI(func(msg ...any) {})

	handler := &mockHandler{name: "Deploy"}
	section := &headlessSection{Title: "DEPLOY"}

	tui.AddHandler(handler, "#FF00FF", section)

	// Dispatch by handler name
	dispatched := tui.DispatchAction("Deploy", "")
	if !dispatched {
		t.Fatal("DispatchAction should have found handler by name")
	}

	// Wait for goroutine to execute
	waitForExecute := make(chan bool, 1)
	go func() {
		for i := 0; i < 100; i++ {
			handler.mu.Lock()
			count := handler.executeCount
			handler.mu.Unlock()
			if count == 1 {
				waitForExecute <- true
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		waitForExecute <- false
	}()

	select {
	case success := <-waitForExecute:
		if !success {
			t.Error("executeCount was never set to 1")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Execute to be called")
	}
}

// headlessSection for testing
type headlessSection struct {
	Title string
}

func (s *headlessSection) GetTitle() string { return s.Title }
