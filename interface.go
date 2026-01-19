package app

import (
	"time"
)

type TuiInterface interface {
	NewTabSection(title, description string) any // returns *tabSection
	AddHandler(Handler any, timeout time.Duration, color string, tabSection any)
	Start(syncWaitGroup ...any) // syncWaitGroup is optional
	RefreshUI()
	ReturnFocus() error       // returns focus to main UI
	SetActiveTab(section any) // sets the active tab section
}

// BrowserInterface defines the browser operations needed by the app.
// Implementations: devbrowser.DevBrowser (production), MockBrowser (tests)
type BrowserInterface interface {
	Reload() error
	AutoStart()
	SetLog(f func(message ...any))
}
