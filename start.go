package app

import (
	"os"
	"sync"

	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/client"
	"github.com/tinywasm/devbrowser"
	"github.com/tinywasm/devwatch"
	"github.com/tinywasm/goflare"
	"github.com/tinywasm/kvdb"
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
	tui           TuiInterface // Interface defined in GOLITE, not DevTUI
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

	// MCP server for shutdown (stored as any to avoid import cycles)
	mcpServer any

	// Test hooks
	pendingBrowserReload func() error
}

// Start is called from main.go with UI passed as parameter
// CRITICAL: UI instance created in main.go, passed here as interface
func Start(rootDir string, logger func(messages ...any), ui TuiInterface, exitChan chan bool) {

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
	h.AddSectionBUILD()
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

	// Auto-configure VS Code MCP integration (silent, non-blocking)
	ConfigureVSCodeMCP()

	SetActiveHandler(h)

	var wg sync.WaitGroup
	wg.Add(4) // UI, server, watcher, and MCP server

	// Start MCP HTTP server on port 3030
	// Now safe because h.config and handlers are initialized
	go func() {
		defer wg.Done()
		h.ServeMCP()
	}()

	// Start the UI (passed from main.go)
	go h.tui.Start(&wg)

	// Start server
	go h.serverHandler.StartServer(&wg)

	// Start file watcher
	go h.watcher.FileWatcherStart(&wg)

	wg.Wait()
}
