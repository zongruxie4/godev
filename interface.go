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
