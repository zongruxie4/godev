package app

import (
	"net/http"
	"sync"

	"github.com/tinywasm/kvdb"
	"github.com/tinywasm/mcpserve"
)

type DB interface {
	kvdb.KVStore
}

type TuiInterface interface {
	NewTabSection(title, description string) any // returns *tabSection
	AddHandler(Handler any, color string, tabSection any)
	Start(syncWaitGroup ...any) // syncWaitGroup is optional
	RefreshUI()
	ReturnFocus() error       // returns focus to main UI
	SetActiveTab(section any) // sets the active tab section

	// GetHandlerStates returns the current handler state as JSON bytes.
	// Format: []StateEntry (devtui wire format).
	// HeadlessTUI: populated by AddHandler calls (daemon mode).
	// DevTUI: returns nil (client mode, not a state server).
	GetHandlerStates() []byte

	// DispatchAction routes a remote action to the handler registered for that key.
	// Returns true if a registered handler handled it; false means caller must handle.
	// HeadlessTUI: iterates handlers slice built in AddHandler, matches by shortcut key.
	// DevTUI: always returns false (actions are sent to daemon, not dispatched locally).
	DispatchAction(key, value string) bool
}

// BrowserInterface defines the browser operations needed by the app.
// Implementations: devbrowser.DevBrowser (production), MockBrowser (tests)
type BrowserInterface interface {
	Reload() error
	OpenBrowser(port string, https bool)
	SetLog(f func(message ...any))
	GetMCPToolsMetadata() []mcpserve.ToolMetadata
}

// ServerInterface is the common contract for all server backends.
// Implemented by: tinywasm/server.ServerHandler, tinywasm/wasi.WasiServer
type ServerInterface interface {
	// Lifecycle
	StartServer(wg *sync.WaitGroup)
	StopServer() error
	RestartServer() error
	// devwatch.FilesEventHandler
	NewFileEvent(fileName, extension, filePath, event string) error
	UnobservedFiles() []string
	SupportedExtensions() []string
	MainInputFileRelativePath() string
	// TUI (devtui.HandlerEdit)
	Name() string
	Label() string
	Value() string
	Change(v string)
	RefreshUI()
	// Route Registration
	RegisterRoutes(fn func(*http.ServeMux))
}

// ServerFactory creates and configures the concrete server.
// Routes and callbacks are NOT passed here — they are registered directly
// on the returned ServerInterface after InitBuildHandlers creates them.
type ServerFactory func(exitChan chan bool, ui TuiInterface, browser BrowserInterface) ServerInterface
