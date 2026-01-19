package app

import (
	"sync"
	"time"

	"github.com/tinywasm/devwatch"
)

// ActiveHandler is set when Start is called so tests can access the live Handler
var ActiveHandler *Handler
var ActiveHandlerMu sync.RWMutex

// SetInitialBrowserReloadFunc sets the Browser reload test hook thread-safely
func SetInitialBrowserReloadFunc(f func() error) {
	initialBrowserReloadMu.Lock()
	defer initialBrowserReloadMu.Unlock()
	initialBrowserReloadFunc = f
}

// GetInitialBrowserReloadFunc gets the Browser reload test hook thread-safely
func GetInitialBrowserReloadFunc() func() error {
	initialBrowserReloadMu.RLock()
	defer initialBrowserReloadMu.RUnlock()
	return initialBrowserReloadFunc
}

var initialBrowserReloadFunc func() error
var initialBrowserReloadMu sync.RWMutex

// SetActiveHandler sets the global Handler instance thread-safely
func SetActiveHandler(h *Handler) {
	ActiveHandlerMu.Lock()
	defer ActiveHandlerMu.Unlock()
	ActiveHandler = h
}

// GetActiveHandler gets the global Handler instance thread-safely
func GetActiveHandler() *Handler {
	ActiveHandlerMu.RLock()
	defer ActiveHandlerMu.RUnlock()
	return ActiveHandler
}

// WaitForActiveHandler waits until the Handler is initialized or timeout
func WaitForActiveHandler(timeout time.Duration) *Handler {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h := GetActiveHandler()
		if h != nil {
			return h
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// WaitWatcherReady waits until the Watcher is initialized or timeout
func WaitWatcherReady(timeout time.Duration) *devwatch.DevWatch {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h := GetActiveHandler()
		if h != nil && h.Watcher != nil {
			return h.Watcher
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// SetWatcherBrowserReload sets the function that DevWatch will call to reload
// the Browser. If the Watcher is already created it updates it immediately,
// otherwise it stores it in the Handler so AddSectionBUILD can apply it.
func SetWatcherBrowserReload(f func() error) {
	ActiveHandlerMu.Lock()
	defer ActiveHandlerMu.Unlock()

	if ActiveHandler == nil {
		return
	}
	if ActiveHandler.Watcher != nil {
		ActiveHandler.Watcher.BrowserReload = f
		return
	}
	ActiveHandler.PendingBrowserReload = f
}
