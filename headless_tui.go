package app

import (
	"fmt"
	"sync"
)

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
}

// NewHeadlessTUI creates a new HeadlessTUI
func NewHeadlessTUI(logger func(messages ...any)) *HeadlessTUI {
	return &HeadlessTUI{logger: logger}
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
