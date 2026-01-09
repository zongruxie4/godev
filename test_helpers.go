package app

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/tinywasm/devflow"
	"github.com/tinywasm/devwatch"
)

// NewTestHandler creates a handler configured for testing.
// It initializes all required dependencies including goHandler.
func NewTestHandler(rootDir string) *handler {
	git, _ := devflow.NewGit()
	gh, _ := devflow.NewGo(git)
	gh.SetRootDir(rootDir)
	git.SetRootDir(rootDir)

	return &handler{
		config:    NewConfig(rootDir, func(...any) {}),
		goHandler: gh,
	}
}

// SafeBuffer is a thread-safe buffer for capturing logs in tests
type SafeBuffer struct {
	mu           sync.Mutex
	buf          bytes.Buffer
	messageLines []string
}

func (s *SafeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *SafeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func (s *SafeBuffer) Log(messages ...any) {
	s.LogReturn(messages...)
}

func (s *SafeBuffer) LogReturn(messages ...any) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msg string
	for i, m := range messages {
		if i > 0 {
			msg += " "
		}
		msg += fmt.Sprint(m)
	}
	s.buf.WriteString(msg + "\n")
	s.messageLines = append(s.messageLines, msg)
	return msg
}

func (s *SafeBuffer) Lines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Return a copy to avoid races on the slice itself
	dst := make([]string, len(s.messageLines))
	copy(dst, s.messageLines)
	return dst
}

func (s *SafeBuffer) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messageLines)
}

func (s *SafeBuffer) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Reset()
	s.messageLines = nil
}

func (s *SafeBuffer) LastLog() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.messageLines) == 0 {
		return ""
	}
	return s.messageLines[len(s.messageLines)-1]
}

// activeHandler is set when Start is called so tests can access the live handler
var activeHandler *handler
var activeHandlerMu sync.RWMutex

// TestMode disables browser auto-start when running tests
var TestMode bool

// initialBrowserReloadFunc allows validating browser reloads without race conditions
var initialBrowserReloadFunc func() error
var initialBrowserReloadMu sync.RWMutex

// SetInitialBrowserReloadFunc sets the browser reload test hook thread-safely
func SetInitialBrowserReloadFunc(f func() error) {
	initialBrowserReloadMu.Lock()
	defer initialBrowserReloadMu.Unlock()
	initialBrowserReloadFunc = f
}

// GetInitialBrowserReloadFunc gets the browser reload test hook thread-safely
func GetInitialBrowserReloadFunc() func() error {
	initialBrowserReloadMu.RLock()
	defer initialBrowserReloadMu.RUnlock()
	return initialBrowserReloadFunc
}

// SetActiveHandler sets the global handler instance thread-safely
func SetActiveHandler(h *handler) {
	activeHandlerMu.Lock()
	defer activeHandlerMu.Unlock()
	activeHandler = h
}

// GetActiveHandler gets the global handler instance thread-safely
func GetActiveHandler() *handler {
	activeHandlerMu.RLock()
	defer activeHandlerMu.RUnlock()
	return activeHandler
}

// WaitForActiveHandler waits until the handler is initialized or timeout
func WaitForActiveHandler(timeout time.Duration) *handler {
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

// WaitWatcherReady waits until the watcher is initialized or timeout
func WaitWatcherReady(timeout time.Duration) *devwatch.DevWatch {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h := GetActiveHandler()
		if h != nil && h.watcher != nil {
			return h.watcher
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// SetWatcherBrowserReload sets the function that DevWatch will call to reload
// the browser. If the watcher is already created it updates it immediately,
// otherwise it stores it in the handler so AddSectionBUILD can apply it.
func SetWatcherBrowserReload(f func() error) {
	activeHandlerMu.Lock()
	defer activeHandlerMu.Unlock()

	if activeHandler == nil {
		return
	}
	if activeHandler.watcher != nil {
		activeHandler.watcher.BrowserReload = f
		return
	}
	activeHandler.pendingBrowserReload = f
}

// EnableDebugWatchEvents switches to debug mode for detailed event logging
func EnableDebugWatchEvents() {
	h := GetActiveHandler()
	if h != nil && h.watcher != nil {
		// We can't easily switch the running watchEvents, but we can log more
		fmt.Println("Debug watch events requested (limited implementation)")
	}
}
