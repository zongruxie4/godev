package app

import (
	"os"
	"sync"
	"time"

	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/client"
	"github.com/tinywasm/devbrowser"
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

	db, err := kvdb.New(".env", logger, FileStore{})
	if err != nil {
		logger("Failed to initialize database:", err)
		return
	}

	h := &handler{
		frameworkName: "TINYWASM",
		rootDir:       rootDir,
		tui:           ui, // UI passed from main.go
		exitChan:      exitChan,

		pendingBrowserReload: InitialBrowserReloadFunc,
		db:                   db,
	}

	// Validate directory
	homeDir, _ := os.UserHomeDir()
	if rootDir == homeDir || rootDir == "/" {
		logger("Cannot run tinywasm in user root directory. Please run in a Go project directory")
		return
	}

	// ADD SECTIONS using the passed UI interface
	// CRITICAL: Initialize sections BEFORE starting goroutines
	// This ensures h.config, h.wasmClient, etc. are set before ServeMCP() tries to use them
	sectionBuild := h.AddSectionBUILD()
	h.AddSectionDEPLOY()

	// Apply persisted work modes
	if h.db != nil {
		if val, err := h.db.Get(StoreKeyBuildModeOnDisk); err == nil && val != "" {
			isDisk := (val == "true")
			h.wasmClient.SetBuildOnDisk(isDisk)
			h.assetsHandler.SetBuildOnDisk(isDisk)
			h.serverHandler.SetBuildOnDisk(isDisk)
		} else {
			// Default to false (In-Memory) as requested
			h.wasmClient.SetBuildOnDisk(false)
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

	var wg sync.WaitGroup
	wg.Add(4) // UI, server, watcher, and MCP server

	// Start MCP HTTP server on the configured port
	go func() {
		defer wg.Done()
		h.mcp.Serve()
	}()

	// Start the UI (passed from main.go)
	go h.tui.Start(&wg)

	// Start server (blocking, so run in goroutine)
	go func() {
		// StartServer blocks until exit, so we usually run it here.
		// However, to ensure browser opens AFTER server is ready, we need a way to know when it's ready.
		// Since StartServer doesn't notify readiness before blocking, we rely on the fact that
		// http.ListenAndServe is called asynchronously inside InMemoryStrategy, but Start blocks on ExitChan.
		// BUT: externalStrategy.Start compiles and runs, then blocks on ExitChan? No, wait.

		// Let's look at strategies.go again.
		// InMemory: blocks on <-ExitChan. Server runs in `go func()`. So actually server IS ready almost immediately.
		// External: compiles then runs.

		// So we can run StartServer and AutoStart in parallel if StartServer blocks?
		// No, if StartServer blocks, we can't run code after it in the same goroutine unless StartServer returns quickly.
		// But inMemoryStrategy.Start blocks on ExitChan! so it DOES NOT return until app exit.

		// FIX: We must run StartServer in a goroutine, and AutoStart in another (or same, but before blocking?)
		// Actually, since StartServer blocks until exit, we can't put AutoStart AFTER it in the same serial flow.

		// However, AutoStart needs server to be ready.
		// Best approach:
		// 1. Start Server in goroutine.
		// 2. Sleep briefly? Or just run AutoStart in another goroutine (it will retry? No retries in OpenBrowser).

		// Let's rely on the fact that OpenBrowser has retries? No, it doesn't.
		// But we can just launch AutoStart in a separate goroutine with a small delay if needed,
		// OR since `go h.serverHandler.StartServer` runs immediately, and `http.Server` starts quickly...

		h.serverHandler.StartServer(&wg)
	}()

	// Auto-open browser (run in separate goroutine to not block main flow, and give server a moment)
	// Skip in TestMode to prevent browser from opening during tests
	if !TestMode {
		go func() {
			// Give the server a moment to initialize ports (especially for external strategy compilation)
			// strictly speaking we should listen to a 'Ready' event, but for now this restores functionality
			// preventing the deadlock.
			time.Sleep(100 * time.Millisecond)
			h.browser.AutoStart()
		}()
	}

	// Start file watcher (blocking, so run in goroutine)
	go h.watcher.FileWatcherStart(&wg)

	wg.Wait()
}

// Browser returns the DevBrowser instance for external access (e.g., tests)
func (h *handler) Browser() *devbrowser.DevBrowser {
	return h.browser
}
