package app

import (
	"time"
)

// ============================================================================
// UI INTERFACES - TINYWASM defines its own interfaces, NO DevTUI import
// ============================================================================

// TuiInterface defines the minimal UI interface needed by TINYWASM.
// This interface is implemented by DevTUI but TINYWASM doesn't know that.
// TINYWASM never imports DevTUI package.
type TuiInterface interface {
	NewTabSection(title, description string) any // returns *tabSection
	AddHandler(handler any, timeout time.Duration, color string, tabSection any)
	Start(syncWaitGroup ...any) // syncWaitGroup is optional
	RefreshUI()
	ReturnFocus() error // returns focus to main UI
}
