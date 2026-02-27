package app

import (
	"fmt"
	"os"
	"sync"

	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/client"
	"github.com/tinywasm/deploy"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/devwatch"
	"github.com/tinywasm/mcpserve"
)

// TestMode disables browser auto-start when running tests
var TestMode bool

// Handler contains application state and dependencies
// CRITICAL: This struct does NOT import DevTUI
type Handler struct {
	FrameworkName string // eg: "TINYWASM", "DEVGO", etc.
	DevMode       bool   // True if running in development mode (read from DB)
	RootDir       string
	Config        *Config
	Tui           TuiInterface // Interface defined in TINYWASM, not DevTUI
	ExitChan      chan bool
	Logger        func(messages ...any) // Main logger for passing to components

	DB DB // Key-value store interface

	// Build dependencies
	Server        ServerInterface
	serverFactory ServerFactory

	AssetsHandler *assetmin.AssetMin
	GitHandler    devflow.GitClient
	GoHandler     *devflow.Go
	GoNew         *devflow.GoNew
	WasmClient    *client.WasmClient
	Watcher       *devwatch.DevWatch
	Browser       BrowserInterface
	GitHubAuth    any

	// Deploy dependencies
	DeployManager *deploy.Daemon

	// Lifecycle management
	startOnce        sync.Once
	SectionBuild     any // Store reference to build tab
	SectionDeploy    any // Store reference to deploy tab
	RestartRequested bool

	// MCP Handler for LLM integration
	MCP *mcpserve.Handler

	// GoMod Handler
	GoModHandler devflow.GoModInterface
}

func (h *Handler) SetBrowser(b BrowserInterface) {
	h.Browser = b
}

func (h *Handler) SetServerFactory(f ServerFactory) {
	h.serverFactory = f
}

// CheckDevMode checks the DB for "dev_mode" key and sets the DevMode field
// CheckDevMode checks the DB for "dev_mode" key and sets the DevMode field
func (h *Handler) CheckDevMode() {
	if h.DB != nil {
		val, err := h.DB.Get("dev_mode")
		// Default to true if not found (err != nil), empty, or explicitly "true"
		if err != nil || val == "" || val == "true" {
			h.DevMode = true
			h.DB.Set("dev_mode", "true")
		}
	}
}

// Start is called from main.go with UI, Browser and DB passed as parameters
// CRITICAL: UI, Browser and DB instances created in main.go, passed here as interfaces
// mcpToolHandlers: optional external Handlers that implement GetMCPToolsMetadata() for MCP tool discovery
func Start(startDir string, logger func(messages ...any), ui TuiInterface, browser BrowserInterface, db DB, ExitChan chan bool, serverFactory ServerFactory, githubAuth any, gitHandler devflow.GitClient, goModHandler devflow.GoModInterface, headless bool, clientMode bool, mcpToolHandlers ...mcpserve.ToolProvider) bool {

	// Initialize Go Handler
	GoHandler, _ := devflow.NewGo(gitHandler)
	GoHandler.SetRootDir(startDir)

	h := &Handler{
		FrameworkName: "TINYWASM",
		RootDir:       startDir,
		Tui:           ui, // UI passed from main.go
		ExitChan:      ExitChan,
		Logger:        logger,

		DB:            db,
		serverFactory: serverFactory,
		GitHandler:    gitHandler,
		GoHandler:     GoHandler,
		Browser:       browser,
		GitHubAuth:    githubAuth,
		GoModHandler:  goModHandler,
	}

	// Wrap logger to also broadcast to MCP if initialized.
	// This ensures that all components using h.Logger will broadcast to SSE.
	h.Logger = func(messages ...any) {
		logger(messages...)
		if h.MCP != nil {
			h.MCP.PublishLog(fmt.Sprint(messages...))
		}
	}

	// Check if we are in dev mode
	h.CheckDevMode()

	// Wire gitignore notification
	if !TestMode {
		gitHandler.SetShouldWrite(h.IsInitializedProject)
	}

	// Validate directory
	homeDir, _ := os.UserHomeDir()
	if startDir == homeDir || startDir == "/" {
		logger("Cannot run tinywasm in user root directory. Please run in a Go project directory")
		return false
	}

	var wg sync.WaitGroup
	wg.Add(1) // UI goroutine; MCP added below only in standalone (non-headless) mode

	// ADD SECTIONS using the passed UI interface
	// CRITICAL: Initialize sections BEFORE starting lifecycle
	h.SectionBuild = h.AddSectionBUILD()
	h.AddSectionDEPLOY()

	// Register GitHubAuth in TUI FIRST (so it gets the TUI logger)
	// This must happen BEFORE starting the auth Future
	if h.GitHubAuth != nil {
		h.Tui.AddHandler(h.GitHubAuth, "#6e40c9", h.SectionBuild) // Purple for GitHub
	}

	// Early return for clientMode: we just needed to construct the TUI sections.
	// We don't want to run Github auth, GoNew, watchers, or the local MCP server in the client.
	if clientMode {
		var clientWg sync.WaitGroup
		clientWg.Add(1)
		go h.Tui.Start(&clientWg)
		clientWg.Wait()
		return false
	}

	// NOW start GitHub auth in background (after TUI registration)
	// This ensures the TUI logger is set before auth messages are sent
	githubFuture := devflow.NewFuture(func() (any, error) {
		if githubAuth != nil {
			if auth, ok := githubAuth.(devflow.GitHubAuthenticator); ok {
				return devflow.NewGitHub(logger, auth)
			}
		}
		return devflow.NewGitHub(logger)
	})

	// Initialize GoNew orchestrator with the future
	GoNew := devflow.NewGoNew(gitHandler, githubFuture, GoHandler)
	GoNew.SetLog(logger)
	h.GoNew = GoNew
	// Prevents goroutine leak: drain future when app exits so the
	// internal goroutine can proceed to close(f.done) and exit.
	go func() {
		select {
		case <-ExitChan:
			<-githubFuture.Ready() // unblock the future's sender goroutine
		case <-githubFuture.Ready():
			// already resolved, nothing to do
		}
	}()

	if !h.IsPartOfProject() {
		sectionWizard := h.AddSectionWIZARD(func() {
			h.OnProjectReady(&wg)
		})
		h.Tui.SetActiveTab(sectionWizard)
	} else {
		h.OnProjectReady(&wg)
	}

	if !headless {
		// Standalone mode: create and serve the project-level MCP on port 3030.
		// In headless/daemon-sub-project mode this is skipped to avoid a port conflict
		// with the already-running global daemon MCP.
		mcpPort := "3030"
		if p := os.Getenv("TINYWASM_MCP_PORT"); p != "" {
			mcpPort = p
		}
		mcpConfig := mcpserve.Config{
			Port:          mcpPort,
			ServerName:    "TinyWasm - Full-stack Go+WASM Dev Environment (Server, WASM, Assets, Browser, Deploy)",
			ServerVersion: "1.0.0",
			AppName:       "tinywasm",
		}
		toolHandlers := []mcpserve.ToolProvider{}
		toolHandlers = append(toolHandlers, h)
		if h.WasmClient != nil {
			toolHandlers = append(toolHandlers, h.WasmClient)
		}
		if h.Browser != nil {
			toolHandlers = append(toolHandlers, h.Browser)
		}
		toolHandlers = append(toolHandlers, mcpToolHandlers...)
		h.MCP = mcpserve.NewHandler(mcpConfig, toolHandlers, h.Tui, h.ExitChan)
		h.Tui.AddHandler(h.MCP, colorOrangeLight, h.SectionBuild)
		h.MCP.ConfigureIDEs()
		SetActiveHandler(h)

		// Start MCP HTTP server
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.MCP.Serve()
		}()

		// Start the UI
		go h.Tui.Start(&wg)
	} else {
		// Headless mode: no MCP, no UI. Keep alive until ExitChan is closed.
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ExitChan
		}()
		wg.Done() // satisfy the initial wg.Add(1) for UI
	}

	wg.Wait()
	return h.RestartRequested
}
