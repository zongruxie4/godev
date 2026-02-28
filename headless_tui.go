package app

import (
	"encoding/json"
	"fmt"
	"sync"
)

// handlerType constants — must match devtui.HandlerType* iota values.
// If devtui/anyHandler.go iota changes, update these in lockstep.
const (
	htDisplay     = 0
	htEdit        = 1
	htExecution   = 2
	htInteractive = 3
)

// capturedHandler holds everything HeadlessTUI knows about one registered handler.
// The handler reference is retained so GetHandlerStates() reads Value()/Label()
// dynamically — never stale, always reflects current handler state.
type capturedHandler struct {
	tabTitle     string
	handlerName  string
	handlerColor string
	handlerType  int          // htDisplay/htEdit/htExecution/htInteractive
	key          string       // shortcut key (empty if none)
	action       func(string) // called on DispatchAction
	handler      any          // original reference for dynamic Value()/Label() reads
}

// headlessSection is a minimal tab section for HeadlessTUI.
// It stores the title so AddHandler can route logs to the correct SSE tab.
type headlessSection struct {
	Title string
}

// GetTitle returns the section title, used by AddHandler for SSE routing.
func (s *headlessSection) GetTitle() string { return s.Title }

// HeadlessTUI implements TuiInterface for headless operation (Daemon mode)
type HeadlessTUI struct {
	logger   func(messages ...any)
	RelayLog func(tabTitle, handlerName, color, msg string) // optional: relay to daemon SSE
	handlers []capturedHandler                                // populated by AddHandler
	mu       sync.RWMutex
}

// NewHeadlessTUI creates a new HeadlessTUI
func NewHeadlessTUI(logger func(messages ...any)) *HeadlessTUI {
	return &HeadlessTUI{logger: logger, handlers: []capturedHandler{}}
}

// NewTabSection returns a headlessSection with an accessible title for log routing.
func (t *HeadlessTUI) NewTabSection(title, description string) any {
	return &headlessSection{Title: title}
}

// AddHandler injects a logger into each component so their logs reach the daemon SSE relay.
// In daemon mode devtui is absent, so this method replicates its logger injection logic.
func (t *HeadlessTUI) AddHandler(handler any, color string, tabSection any) {
	// Extract tab title for SSE routing
	tabTitle := "BUILD"
	type titleGetter interface{ GetTitle() string }
	if ts, ok := tabSection.(titleGetter); ok {
		tabTitle = ts.GetTitle()
	}

	// Extract handler name for SSE metadata
	type namer interface{ Name() string }
	handlerName := "HANDLER"
	if n, ok := handler.(namer); ok {
		handlerName = n.Name()
	}

	// Inject a relay logger into the component (mirrors devtui.registerLoggableHandler)
	type logSetter interface{ SetLog(func(...any)) }
	if s, ok := handler.(logSetter); ok {
		s.SetLog(func(messages ...any) {
			msg := fmt.Sprint(messages...)
			if t.logger != nil {
				t.logger(msg)
			}
			if t.RelayLog != nil {
				t.RelayLog(tabTitle, handlerName, color, msg)
			}
		})
	}

	// Capture handler state and actions
	t.mu.Lock()
	defer t.mu.Unlock()

	ch := capturedHandler{
		tabTitle:     tabTitle,
		handlerName:  handlerName,
		handlerColor: color,
		handler:      handler,
	}

	type display interface{ Value() string }
	type edit interface {
		Value() string
		Change(string)
	}
	type execution interface {
		Execute()
		Shortcut() string
	}
	type interactive interface {
		Execute(string)
		Shortcut() string
	}

	// Determine type and bind action/shortcut
	switch h := handler.(type) {
	case interactive:
		ch.handlerType = htInteractive
		ch.key = h.Shortcut()
		ch.action = h.Execute
	case execution:
		ch.handlerType = htExecution
		ch.key = h.Shortcut()
		ch.action = func(string) { h.Execute() }
	case edit:
		ch.handlerType = htEdit
		ch.action = h.Change
	case display:
		ch.handlerType = htDisplay
	default:
		// Unknown type, default to display
		ch.handlerType = htDisplay
	}

	t.handlers = append(t.handlers, ch)
}

// GetHandlerStates reads current state dynamically from each handler reference.
// JSON tags match devtui.StateEntry exactly — this is the published wire contract.
func (t *HeadlessTUI) GetHandlerStates() []byte {
	t.mu.RLock()
	defer t.mu.RUnlock()

	type labeler interface{ Label() string }
	type valuer interface{ Value() string }

	type stateEntry struct {
		TabTitle     string `json:"tab_title"`
		HandlerName  string `json:"handler_name"`
		HandlerColor string `json:"handler_color"`
		HandlerType  int    `json:"handler_type"`
		Label        string `json:"label"`
		Value        string `json:"value"`
		Shortcut     string `json:"shortcut"`
	}

	entries := make([]stateEntry, 0, len(t.handlers))
	for _, h := range t.handlers {
		e := stateEntry{
			TabTitle:     h.tabTitle,
			HandlerName:  h.handlerName,
			HandlerColor: h.handlerColor,
			HandlerType:  h.handlerType,
			Shortcut:     h.key,
		}
		if l, ok := h.handler.(labeler); ok {
			e.Label = l.Label()
		}
		if v, ok := h.handler.(valuer); ok {
			e.Value = v.Value()
		}
		entries = append(entries, e)
	}
	data, _ := json.Marshal(entries)
	return data
}

// DispatchAction routes a remote action to the handler registered for that key.
func (t *HeadlessTUI) DispatchAction(key, value string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, h := range t.handlers {
		if h.key != "" && h.key == key && h.action != nil {
			go h.action(value) // non-blocking
			return true
		}
	}
	return false
}

// Start does nothing in headless mode (no UI loop)
func (t *HeadlessTUI) Start(syncWaitGroup ...any) {
	// Just signal done if waitgroup provided
	for _, wg := range syncWaitGroup {
		if w, ok := wg.(*sync.WaitGroup); ok {
			w.Done()
		}
	}
}

// RefreshUI does nothing
func (t *HeadlessTUI) RefreshUI() {}

// ReturnFocus does nothing
func (t *HeadlessTUI) ReturnFocus() error {
	return nil
}

// SetActiveTab does nothing
func (t *HeadlessTUI) SetActiveTab(section any) {}
