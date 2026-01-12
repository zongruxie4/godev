package app

import (
	"sync"
	"time"
)

type mockTUI struct {
	logger func(...any)
}

func newUiMockTest(opt ...any) *mockTUI {
	m := &mockTUI{}
	if len(opt) > 0 {
		if l, ok := opt[0].(func(...any)); ok {
			m.logger = l
		}
	}
	return m
}

func (m *mockTUI) NewTabSection(title, description string) any {
	return nil
}

func (m *mockTUI) AddHandler(handler any, timeout time.Duration, color string, tabSection any) {
	// Mimic DevTUI's behavior: call SetLog if handler implements Loggable
	type Loggable interface {
		Name() string
		SetLog(logger func(message ...any))
	}

	if loggable, ok := handler.(Loggable); ok {
		logFunc := func(message ...any) {
			if m.logger != nil {
				m.logger(message...)
			}
		}
		loggable.SetLog(logFunc)
	}
}

func (m *mockTUI) AddLogger(name string, enableTracking bool, color string, tabSection any) func(message ...any) {
	return func(messages ...any) {
		if m.logger != nil {
			m.logger(messages...)
		}
	}
}

func (m *mockTUI) Start(syncWaitGroup ...any) {
	if len(syncWaitGroup) > 0 {
		if wg, ok := syncWaitGroup[0].(*sync.WaitGroup); ok {
			defer wg.Done()
		}
	}
	// Mimic blocking behavior if needed, or just return.
	// Real TUI blocks?
	// If real TUI blocks, we should probably block until exit?
	// But mockTUI is simple.
	// If we just Done() and return, wg decrements.
	// Start continues waiting for others.
}

func (m *mockTUI) RefreshUI() {
	// no-op
}

func (m *mockTUI) ReturnFocus() error {
	return nil
}

func (m *mockTUI) SetActiveTab(section any) {
	// no-op
}
