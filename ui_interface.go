package godev

import (
	"time"
)

// ============================================================================
// UI INTERFACES - GODEV defines its own interfaces, NO DevTUI import
// ============================================================================

// TuiInterface defines the minimal UI interface needed by GODEV.
// This interface is implemented by DevTUI but GODEV doesn't know that.
// GODEV never imports DevTUI package.
type TuiInterface interface {
	NewTabSection(title, description string) any // returns *tabSection
	AddHandler(handler any, timeout time.Duration, color string, tabSection any)
	AddLogger(name string, enableTracking bool, color string, tabSection any) func(message ...any)
	Start(syncWaitGroup ...any) // syncWaitGroup is optional
	ReturnFocus() error         // returns focus to main UI
}
