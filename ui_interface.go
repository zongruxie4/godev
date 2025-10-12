package godev

import (
	"sync"
	"time"
)

// ============================================================================
// UI INTERFACES - GODEV defines its own interfaces, NO DevTUI import
// ============================================================================

// TuiInterface defines the minimal UI interface needed by GODEV.
// This interface is implemented by DevTUI but GODEV doesn't know that.
// GODEV never imports DevTUI package.
type TuiInterface interface {
	// NewTabSection creates a new tab section
	NewTabSection(title, description string) TabSectionInterface

	// Start initializes and runs the UI
	Start(wg *sync.WaitGroup)
}

// TabSectionInterface defines the minimal tab section interface.
type TabSectionInterface interface {
	// AddHandler registers any type of handler
	AddHandler(handler any, timeout time.Duration, color string)

	// AddLogger creates a logger function for this tab section
	// Returns anonymous function for simple usage: log("message")
	AddLogger(name string, enableTracking bool, color string) func(message ...any)
}