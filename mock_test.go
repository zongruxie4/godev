package golite

import "time"

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
	// no-op
}

func (m *mockTUI) AddLogger(name string, enableTracking bool, color string, tabSection any) func(message ...any) {
	return func(messages ...any) {
		if m.logger != nil {
			m.logger(messages...)
		}
	}
}

func (m *mockTUI) Start(syncWaitGroup ...any) {
	// no-op
}

func (m *mockTUI) ReturnFocus() error {
	return nil
}
