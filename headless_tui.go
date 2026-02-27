package app

import (
	"sync"
)

// HeadlessTUI implements TuiInterface for headless operation (Daemon mode)
type HeadlessTUI struct {
	logger func(messages ...any)
}

// NewHeadlessTUI creates a new HeadlessTUI
func NewHeadlessTUI(logger func(messages ...any)) *HeadlessTUI {
	return &HeadlessTUI{logger: logger}
}

// NewTabSection returns a dummy tab section
func (t *HeadlessTUI) NewTabSection(title, description string) any {
	// Return a dummy object or nil if possible.
	// Since section is usually opaque, any should work.
	return &struct{ Title string }{Title: title}
}

// AddHandler does nothing in headless mode, but logs the registration
func (t *HeadlessTUI) AddHandler(Handler any, color string, tabSection any) {
	// Handlers are still registered for MCP tool discovery,
	// but UI doesn't render them.
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
