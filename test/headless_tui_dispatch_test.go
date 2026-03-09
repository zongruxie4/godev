package test

import (
	"encoding/json"
	"testing"

	"github.com/tinywasm/app"
)

// mockHandler implements all necessary interfaces
type mockHandler struct {
	name      string
	value     string
	label     string
	shortcuts []map[string]string
}

func (m *mockHandler) Name() string                { return m.name }
func (m *mockHandler) Value() string              { return m.value }
func (m *mockHandler) Label() string              { return m.label }
func (m *mockHandler) Execute()                   {}
func (m *mockHandler) Change(v string)            {}
func (m *mockHandler) Shortcuts() []map[string]string { return m.shortcuts }

// TestHeadlessTUI_DispatchFindsHandlerByName verifies DispatchAction finds handlers by name
func TestHeadlessTUI_DispatchFindsHandlerByName(t *testing.T) {
	tui := app.NewHeadlessTUI(func(msg ...any) {})
	handler := &mockHandler{name: "TestEdit", value: "initial", label: "Test Label"}
	section := &headlessSection{Title: "TEST"}
	tui.AddHandler(handler, "#FF0000", section)

	// Dispatch by handler name should be found
	dispatched := tui.DispatchAction("TestEdit", "newvalue")
	if !dispatched {
		t.Fatal("DispatchAction should have found handler by name")
	}
}

// TestHeadlessTUI_DispatchFindsByShortcutKey verifies DispatchAction finds handlers by shortcut key
func TestHeadlessTUI_DispatchFindsByShortcutKey(t *testing.T) {
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

// TestHeadlessTUI_DispatchNotFoundRetursFalse verifies DispatchAction returns false for missing keys
func TestHeadlessTUI_DispatchNotFoundReturnsFalse(t *testing.T) {
	tui := app.NewHeadlessTUI(func(msg ...any) {})
	handler := &mockHandler{name: "Deploy"}
	section := &headlessSection{Title: "DEPLOY"}
	tui.AddHandler(handler, "#FF00FF", section)

	// Dispatch with non-existent key
	dispatched := tui.DispatchAction("NonExistent", "")
	if dispatched {
		t.Fatal("DispatchAction should NOT have found handler for NonExistent key")
	}
}

// headlessSection for testing
type headlessSection struct {
	Title string
}

func (s *headlessSection) GetTitle() string { return s.Title }
