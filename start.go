package app

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/client"
	"github.com/tinywasm/devbrowser"
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

// handler contains application state and dependencies
// CRITICAL: This struct does NOT import DevTUI
type handler struct {
	frameworkName string // eg: "TINYWASM", "DEVGO", etc.
	rootDir       string
	config        *Config
	tui           TuiInterface // Interface defined in TINYWASM, not DevTUI
	exitChan      chan bool

	db Store // Key-value store interface

	// Build dependencies
	serverHandler *server.ServerHandler
	assetsHandler *assetmin.AssetMin
	goHandler     *devflow.Go
	wasmClient    *client.WasmClient
	watcher       *devwatch.DevWatch
	browser       *devbrowser.DevBrowser

	// Deploy dependencies
	deployCloudflare *goflare.Goflare

	// MCP handler for LLM integration
	mcp *mcpserve.Handler

	// Test hooks
	pendingBrowserReload func() error
}

// Start is called from main.go with UI passed as parameter
// CRITICAL: UI instance created in main.go, passed here as interface
// mcpToolHandlers: optional external handlers that implement GetMCPToolsMetadata() for MCP tool discovery
func Start(rootDir string, logger func(messages ...any), ui TuiInterface, exitChan chan bool, mcpToolHandlers ...any) {

	// Initialize Git handler for gitignore management
	gitHandler, _ := devflow.NewGit()
	gitHandler.SetRootDir(rootDir)

	// Initialize Go handler
	goHandler, _ := devflow.NewGo(gitHandler)
	goHandler.SetRootDir(rootDir)

	fileStore := &FileStore{}
	var storeToUse kvdb.Store = fileStore
	if TestMode {
		storeToUse = NewMemoryStore()
	}

	db, err := kvdb.New(".env", logger, storeToUse)
	if err != nil {
		logger("Failed to initialize database:", err)
		return
	}

	h := &handler{
		frameworkName: "TINYWASM",
		rootDir:       rootDir,
		tui:           ui, // UI passed from main.go
		exitChan:      exitChan,

		pendingBrowserReload: GetInitialBrowserReloadFunc(),
		db:                   db,
		goHandler:            goHandler,
	}

	// Wire FileStore guard and gitignore notification (only if not TestMode)
	if !TestMode {
		fileStore.SetShouldWrite(h.isInitializedProject)
		gitHandler.SetShouldWrite(h.isInitializedProject)
		fileStore.SetOnFileCreated(func(path string) {
			if filepath.Base(path) == ".env" {
				gitHandler.GitIgnoreAdd(".env")
			}
		})
	}

	// Validate directory
	homeDir, _ := os.UserHomeDir()
	if rootDir == homeDir || rootDir == "/" {
		logger("Cannot run tinywasm in user root directory. Please run in a Go project directory")
		return
	}

	sectionBuild := h.AddSectionBUILD()
	h.AddSectionDEPLOY()

	var wg sync.WaitGroup
	wg.Add(4) // UI, server, watcher, and MCP server

	var startOnce sync.Once

	// startServices launches the server and browser
	// It is called either immediately (if project exists) or after Wizard (if setup needed)
	startServices := func() {
		startOnce.Do(func() {
			h.tui.SetActiveTab(sectionBuild)

			// Start server (blocking, so run in goroutine)
			go h.serverHandler.StartServer(&wg)

			// Start file watcher (blocking, so run in goroutine)
			go h.watcher.FileWatcherStart(&wg)

			// Auto-open browser (run in separate goroutine to not block main flow)
			// Skip in TestMode to prevent browser from opening during tests
			if !TestMode {
				go func() {
					time.Sleep(100 * time.Millisecond)
					h.browser.AutoStart()
				}()
			}
		})
	}

	// ADD SECTIONS using the passed UI interface
	// CRITICAL: Initialize sections BEFORE starting goroutines
	if !h.isPartOfProject() {
		sectionWizard := h.AddSectionWIZARD(startServices)
		h.tui.SetActiveTab(sectionWizard)
	} else {
		h.tui.SetActiveTab(sectionBuild)
		startServices()
	}

	// Apply persisted work modes
	if h.db != nil {
		if val, err := h.db.Get(StoreKeyBuildModeOnDisk); err == nil && val != "" {
			isDisk := (val == "true")
			h.wasmClient.SetBuildOnDisk(isDisk, true)
			h.assetsHandler.SetBuildOnDisk(isDisk)
			h.serverHandler.SetBuildOnDisk(isDisk)
		} else {
			// Default to false (In-Memory) as requested
			h.wasmClient.SetBuildOnDisk(false, true)
			h.assetsHandler.SetBuildOnDisk(false)
			h.serverHandler.SetBuildOnDisk(false)
		}

		if val, err := h.db.Get(server.StoreKeyExternalServer); err == nil && val != "" {
			isExternal := (val == "true")
			h.serverHandler.SetExternalServerMode(isExternal)
		} else {
			// Default to false (Internal) as requested
			h.serverHandler.SetExternalServerMode(false)
		}
	}

	// Auto-configure IDE MCP integration (silent, non-blocking)
	mcpConfig := mcpserve.Config{
		Port:          "3030",
		ServerName:    "TinyWasm - Full-stack Go+WASM Dev Environment (Server, WASM, Assets, Browser, Deploy)",
		ServerVersion: "1.0.0",
		AppName:       "tinywasm", // Used to generate MCP server ID (e.g., "tinywasm-mcp")
	}
	toolHandlers := []any{}
	if h.wasmClient != nil {
		toolHandlers = append(toolHandlers, h.wasmClient)
	}
	if h.browser != nil {
		toolHandlers = append(toolHandlers, h.browser)
	}
	// Add external MCP tool handlers (e.g., DevTUI for devtui_get_section_logs)
	toolHandlers = append(toolHandlers, mcpToolHandlers...)
	h.mcp = mcpserve.NewHandler(mcpConfig, toolHandlers, h.tui, h.exitChan)
	// Register MCP handler in BUILD section for logging visibility
	h.tui.AddHandler(h.mcp, 0, "#FF9500", sectionBuild) // Orange color for MCP

	h.mcp.ConfigureIDEs()

	SetActiveHandler(h)

	// Start MCP HTTP server on the configured port
	go func() {
		defer wg.Done()
		h.mcp.Serve()
	}()

	// Start the UI (passed from main.go)
	go h.tui.Start(&wg)

	wg.Wait()
}

// Browser returns the DevBrowser instance for external access (e.g., tests)
func (h *handler) Browser() *devbrowser.DevBrowser {
	return h.browser
}
