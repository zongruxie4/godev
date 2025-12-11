package tinywasm

import (
	"time"
)

// ============================================================================
// UI INTERFACES - GOLITE defines its own interfaces, NO DevTUI import
// ============================================================================

// TuiInterface defines the minimal UI interface needed by GOLITE.
// This interface is implemented by DevTUI but GOLITE doesn't know that.
// GOLITE never imports DevTUI package.
type TuiInterface interface {
	NewTabSection(title, description string) any // returns *tabSection
	AddHandler(handler any, timeout time.Duration, color string, tabSection any)
	AddLogger(name string, enableTracking bool, color string, tabSection any) func(message ...any)
	Start(syncWaitGroup ...any) // syncWaitGroup is optional
	RefreshUI()
	ReturnFocus() error // returns focus to main UI
}
