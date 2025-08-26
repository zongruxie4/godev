package godev

import "fmt"

// ActiveHandler is set when Start is called so tests can access the live handler
var ActiveHandler *handler

// SetWatcherBrowserReload sets the function that DevWatch will call to reload
// the browser. If the watcher is already created it updates it immediately,
// otherwise it stores it in the handler so AddSectionBUILD can apply it.
func SetWatcherBrowserReload(f func() error) {
	if ActiveHandler == nil {
		fmt.Println("DEBUG: SetWatcherBrowserReload called but ActiveHandler is nil")
		return
	}
	if ActiveHandler.watcher != nil {
		fmt.Println("DEBUG: SetWatcherBrowserReload applying to existing watcher")
		ActiveHandler.watcher.BrowserReload = f
		return
	}
	fmt.Println("DEBUG: SetWatcherBrowserReload storing as pending")
	ActiveHandler.pendingBrowserReload = f
}

// EnableDebugWatchEvents switches to debug mode for detailed event logging
func EnableDebugWatchEvents() {
	if ActiveHandler != nil && ActiveHandler.watcher != nil {
		// We can't easily switch the running watchEvents, but we can log more
		fmt.Println("DEBUG: Debug watch events requested (limited implementation)")
	}
}
