package app

import (
	"github.com/tinywasm/mcpserve"
)

type TuiInterface interface {
	NewTabSection(title, description string) any // returns *tabSection
	AddHandler(Handler any, color string, tabSection any)
	Start(syncWaitGroup ...any) // syncWaitGroup is optional
	RefreshUI()
	ReturnFocus() error       // returns focus to main UI
	SetActiveTab(section any) // sets the active tab section
}

// BrowserInterface defines the browser operations needed by the app.
// Implementations: devbrowser.DevBrowser (production), MockBrowser (tests)
type BrowserInterface interface {
	Reload() error
	OpenBrowser(port string, https bool)
	SetLog(f func(message ...any))
	GetMCPToolsMetadata() []mcpserve.ToolMetadata
}
