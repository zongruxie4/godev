package app

import (
	"os"
	"path/filepath"
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

type Store interface {
	Get(key string) (string, error)
	Set(key, value string) error
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

	DB Store // Key-value store interface

	// Build dependencies
	ServerHandler *server.ServerHandler
	AssetsHandler *assetmin.AssetMin
	GitHandler    *devflow.Git
	GoHandler     *devflow.Go
	GoNew         *devflow.GoNew
	WasmClient    *client.WasmClient
	Watcher       *devwatch.DevWatch
	Browser       BrowserInterface

	// Deploy dependencies
	DeployCloudflare *goflare.Goflare

	// Lifecycle management
	startOnce     sync.Once
	SectionBuild  any // Store reference to build tab
	SectionDeploy any // Store reference to deploy tab

	// MCP Handler for LLM integration
	MCP *mcpserve.Handler
}

func (h *Handler) SetBrowser(b BrowserInterface) {
	h.Browser = b
}

// Start is called from main.go with UI passed as parameter
// CRITICAL: UI instance created in main.go, passed here as interface
// mcpToolHandlers: optional external Handlers that implement GetMCPToolsMetadata() for MCP tool discovery
func Start(RootDir string, logger func(messages ...any), ui TuiInterface, ExitChan chan bool, mcpToolHandlers ...any) {

	// Initialize Git Handler for gitignore management
	GitHandler, _ := devflow.NewGit()
	GitHandler.SetRootDir(RootDir)

	// Initialize Go Handler
	GoHandler, _ := devflow.NewGo(GitHandler)
	GoHandler.SetRootDir(RootDir)

	fileStore := &FileStore{}
	var storeToUse kvdb.Store = fileStore
	if TestMode {
		storeToUse = NewMemoryStore()
	}

	DB, err := kvdb.New(".env", logger, storeToUse)
	if err != nil {
		logger("Failed to initialize database:", err)
		return
	}

	// Start GitHub auth in background (non-blocking)
	githubFuture := devflow.NewFuture(func() (any, error) {
		return devflow.NewGitHub(logger)
	})

	// Initialize GoNew orchestrator with the future
	GoNew := devflow.NewGoNew(GitHandler, githubFuture, GoHandler)
	GoNew.SetLog(logger)

	h := &Handler{
		FrameworkName: "TINYWASM",
		RootDir:       RootDir,
		Tui:           ui, // UI passed from main.go
		ExitChan:      ExitChan,

		DB:         DB,
		GitHandler: GitHandler,
		GoHandler:  GoHandler,
		GoNew:      GoNew,
		Browser:    GetInitialBrowser(),
	}

	// Wire FileStore guard and gitignore notification (only if not TestMode)
	if !TestMode {
		fileStore.SetShouldWrite(h.IsInitializedProject)
		GitHandler.SetShouldWrite(h.IsInitializedProject)
		fileStore.SetOnFileCreated(func(path string) {
			if filepath.Base(path) == ".env" {
				GitHandler.GitIgnoreAdd(".env")
			}
		})
	}

	// Validate directory
	homeDir, _ := os.UserHomeDir()
	if RootDir == homeDir || RootDir == "/" {
		logger("Cannot run tinywasm in user root directory. Please run in a Go project directory")
		return
	}

	var wg sync.WaitGroup
	wg.Add(4) // UI, server, Watcher, and MCP server

	// ADD SECTIONS using the passed UI interface
	// CRITICAL: Initialize sections BEFORE starting lifecycle
	h.SectionBuild = h.AddSectionBUILD()
	h.AddSectionDEPLOY()

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
	h.Tui.AddHandler(h.MCP, 0, "#FF9500", h.SectionBuild) // Orange color for MCP

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
