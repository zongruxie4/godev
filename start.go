package app

import (
	"os"
	"sync"

	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/client"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/devwatch"
	"github.com/tinywasm/goflare"
	"github.com/tinywasm/kvdb"
	"github.com/tinywasm/mcpserve"
	"github.com/tinywasm/server"
)

type DB interface {
	kvdb.KVStore
}

// TestMode disables browser auto-start when running tests
var TestMode bool

// Handler contains application state and dependencies
// CRITICAL: This struct does NOT import DevTUI
type Handler struct {
	FrameworkName string // eg: "TINYWASM", "DEVGO", etc.
	RootDir       string
	Config        *Config
	Tui           TuiInterface // Interface defined in TINYWASM, not DevTUI
	ExitChan      chan bool
	Logger        func(messages ...any) // Main logger for passing to components

	DB DB // Key-value store interface

	// Build dependencies
	ServerHandler *server.ServerHandler
	AssetsHandler *assetmin.AssetMin
	GitHandler    devflow.GitClient
	GoHandler     *devflow.Go
	GoNew         *devflow.GoNew
	WasmClient    *client.WasmClient
	Watcher       *devwatch.DevWatch
	Browser       BrowserInterface
	GitHubAuth    any

	// Deploy dependencies
	DeployCloudflare *goflare.Goflare

	// Lifecycle management
	// Lifecycle management
	startOnce     sync.Once
	SectionBuild  any // Store reference to build tab
	SectionDeploy any // Store reference to deploy tab

	// MCP Handler for LLM integration
	MCP *mcpserve.Handler

	// GoMod Handler
	GoModHandler devflow.GoModInterface
}

func (h *Handler) SetBrowser(b BrowserInterface) {
	h.Browser = b
}

// Start is called from main.go with UI, Browser and DB passed as parameters
// CRITICAL: UI, Browser and DB instances created in main.go, passed here as interfaces
// mcpToolHandlers: optional external Handlers that implement GetMCPToolsMetadata() for MCP tool discovery
func Start(startDir string, logger func(messages ...any), ui TuiInterface, browser BrowserInterface, db DB, ExitChan chan bool, githubAuth any, gitHandler devflow.GitClient, goModHandler devflow.GoModInterface, mcpToolHandlers ...any) {

	// Initialize Go Handler
	GoHandler, _ := devflow.NewGo(gitHandler)
	GoHandler.SetRootDir(startDir)

	h := &Handler{
		FrameworkName: "TINYWASM",
		RootDir:       startDir,
		Tui:           ui, // UI passed from main.go
		ExitChan:      ExitChan,
		Logger:        logger,

		DB:           db,
		GitHandler:   gitHandler,
		GoHandler:    GoHandler,
		Browser:      browser,
		GitHubAuth:   githubAuth,
		GoModHandler: goModHandler,
	}

	// Wire gitignore notification
	if !TestMode {
		gitHandler.SetShouldWrite(h.IsInitializedProject)
	}

	// Validate directory
	homeDir, _ := os.UserHomeDir()
	if startDir == homeDir || startDir == "/" {
		logger("Cannot run tinywasm in user root directory. Please run in a Go project directory")
		return
	}

	var wg sync.WaitGroup
	wg.Add(4) // UI, server, Watcher, and MCP server

	// ADD SECTIONS using the passed UI interface
	// CRITICAL: Initialize sections BEFORE starting lifecycle
	h.SectionBuild = h.AddSectionBUILD()
	h.AddSectionDEPLOY()

	// Register GitHubAuth in TUI FIRST (so it gets the TUI logger)
	// This must happen BEFORE starting the auth Future
	if h.GitHubAuth != nil {
		h.Tui.AddHandler(h.GitHubAuth, "#6e40c9", h.SectionBuild) // Purple for GitHub
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

	if !h.IsPartOfProject() {
		sectionWizard := h.AddSectionWIZARD(func() {
			h.OnProjectReady(&wg)
		})
		h.Tui.SetActiveTab(sectionWizard)
	} else {
		h.OnProjectReady(&wg)
	}

	// Auto-configure IDE MCP integration (silent, non-blocking)
	mcpConfig := mcpserve.Config{
		Port:          "3030",
		ServerName:    "TinyWasm - Full-stack Go+WASM Dev Environment (Server, WASM, Assets, Browser, Deploy)",
		ServerVersion: "1.0.0",
		AppName:       "tinywasm", // Used to generate MCP server ID (e.g., "tinywasm-MCP")
	}
	toolHandlers := []any{}
	if h.WasmClient != nil {
		toolHandlers = append(toolHandlers, h.WasmClient)
	}
	if h.Browser != nil {
		toolHandlers = append(toolHandlers, h.Browser)
	}
	// Add external MCP tool Handlers (e.g., DevTUI for devtui_get_section_logs)
	toolHandlers = append(toolHandlers, mcpToolHandlers...)
	h.MCP = mcpserve.NewHandler(mcpConfig, toolHandlers, h.Tui, h.ExitChan)
	// Register MCP Handler in BUILD section for logging visibility
	h.Tui.AddHandler(h.MCP, colorOrangeLight, h.SectionBuild) // Orange color for MCP

	h.MCP.ConfigureIDEs()

	SetActiveHandler(h)

	// Start MCP HTTP server on the configured port
	go func() {
		defer wg.Done()
		h.MCP.Serve()
	}()

	// Start the UI (passed from main.go)
	go h.Tui.Start(&wg)

	wg.Wait()
}
