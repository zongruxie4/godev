package app

import (
	"sync"

	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/client"
	"github.com/tinywasm/deploy"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/devwatch"
	"github.com/tinywasm/mcp"
)

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

	// MCP Server for LLM integration (owns /mcp, /logs, /action, /state, /version routes)
	MCP *mcp.Server

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
