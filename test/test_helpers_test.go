package test

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/tinywasm/app"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/devwatch"
)

// NewTestHandler creates a app.Handler configured for testing.
// It initializes all required dependencies including GoHandler.
func NewTestHandler(RootDir string) *app.Handler {
	git, _ := devflow.NewGit()
	gh, _ := devflow.NewGo(git)
	gh.SetRootDir(RootDir)
	git.SetRootDir(RootDir)

	h := &app.Handler{
		Config:    app.NewConfig(RootDir, func(...any) {}),
		GoHandler: gh,
		Keys:      &mockSecretStore{},
	}

	h.SetServerFactory(func() app.ServerInterface {
		return &MockServer{}
	})

	return h
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

// WaitWatcherReady waits until the Watcher is initialized or timeout
func WaitWatcherReady(timeout time.Duration) *devwatch.DevWatch {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h := app.GetActiveHandler()
		if h != nil && h.Watcher != nil {
			return h.Watcher
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// EnableDebugWatchEvents switches to debug mode for detailed event logging
func EnableDebugWatchEvents() {
	h := app.GetActiveHandler()
	if h != nil && h.Watcher != nil {
		// We can't easily switch the running watchEvents, but we can log more
	}
}
