package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/tinywasm/devflow"
	"github.com/tinywasm/mcp"
)

// BootstrapConfig holds dependencies for bootstrapping the application
type BootstrapConfig struct {
	StartDir        string
	McpMode         bool
	Debug           bool
	Version         string // Binary version, used to detect and replace stale daemons
	AppName         string // e.g. "tinywasm" — used in HTTP server version endpoint
	APIKeyPath      string // path to persist API key; empty = no auth (open mode)
	Logger          any    // can be func(...any) or *Logger
	DB              DB
	GitHandler      devflow.GitClient
	GoModHandler    devflow.GoModInterface
	ServerFactory   ServerFactory
	TuiFactory      func(exitChan chan bool, clientMode bool, clientURL, apiKey string) TuiInterface
	BrowserFactory  func(ui TuiInterface, exitChan chan bool) BrowserInterface
	GitHubAuth      any
	McpToolHandlers []mcp.ToolProvider
}

// Bootstrap is the main entry point for the application logic
func Bootstrap(cfg BootstrapConfig) {
	var loggerFunc func(messages ...any)
	if l, ok := cfg.Logger.(func(...any)); ok {
		loggerFunc = l
	} else if l, ok := cfg.Logger.(*Logger); ok {
		loggerFunc = l.Logger
	}
	if loggerFunc == nil {
		loggerFunc = func(messages ...any) {}
	}

	// 1. Check if we should run as Daemon (headless) or Client (TUI)

	if cfg.McpMode {
		// Force Daemon mode
		runDaemon(cfg)
		return
	}

	// Check if port 3030 is open (meaning occupied by another instance)
	if isPortOpen("3030") {
		// Port occupied -> check if the running daemon is the same version
		if cfg.Version != "" && !isDaemonVersionCurrent("3030", cfg.Version) {
			// Stale daemon detected: kill it and start a fresh one
			killDaemon()
			waitForPortFree("3030")

			if err := startDaemonProcess(cfg.StartDir, loggerFunc); err != nil {
				fmt.Printf("Failed to restart daemon: %v\n", err)
				os.Exit(1)
			}
			waitForPortReady("3030")
		}
		runClient(cfg)
	} else {
		// Port free -> Start Daemon in background, then run Client
		if err := startDaemonProcess(cfg.StartDir, loggerFunc); err != nil {
			fmt.Printf("Failed to start daemon: %v\n", err)
			os.Exit(1)
		}
		waitForPortReady("3030")
		runClient(cfg)
	}
}

func runClient(cfg BootstrapConfig) {
	var loggerFunc func(messages ...any)
	if l, ok := cfg.Logger.(func(...any)); ok {
		loggerFunc = l
	} else if l, ok := cfg.Logger.(*Logger); ok {
		loggerFunc = l.Logger
	}
	if loggerFunc == nil {
		loggerFunc = func(messages ...any) {}
	}

	// Client mode
	// Use real TUI with SSE client enabled to receive logs from the daemon
	exitChan := make(chan bool)
	mcpPort := "3030"
	if p := os.Getenv("TINYWASM_MCP_PORT"); p != "" {
		mcpPort = p
	}
	baseURL := "http://localhost:" + mcpPort
	clientURL := baseURL + "/logs"

	// Read API key (daemon already created it on its startup)
	apiKey := readAPIKey(cfg.APIKeyPath)

	// TuiFactory now receives apiKey so devtui can attach auth to /logs SSE
	ui := cfg.TuiFactory(exitChan, true, clientURL, apiKey)

	// Tell the daemon to start (or restart) the project in the current directory.
	// This ensures every `tinywasm` invocation activates the project for its working dir.
	if cfg.StartDir != "" {
		body, err := json.Marshal(map[string]string{"key": "start", "value": cfg.StartDir})
		if err != nil {
			loggerFunc("Error marshaling action body:", err)
		} else {
			req, err := http.NewRequest("POST", baseURL+"/tinywasm/action", bytes.NewReader(body))
			if err != nil {
				loggerFunc("Error creating request:", err)
			} else {
				req.Header.Set("Content-Type", "application/json")
				if apiKey != "" {
					req.Header.Set("Authorization", "Bearer "+apiKey)
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					loggerFunc("Error sending start action to daemon:", err)
				} else {
					resp.Body.Close()
				}
			}
		}
	}

	// In Client Mode, we use Start to orchestrate the tabs without running the backend
	Start(
		cfg.StartDir,
		cfg.Logger,
		ui,
		cfg.BrowserFactory(ui, exitChan), // Browser might be needed for commands
		cfg.DB,
		exitChan,
		cfg.ServerFactory,
		cfg.GitHubAuth,
		cfg.GitHandler,
		cfg.GoModHandler,
		false, // headless (we want UI)
		true,  // clientMode true! skips backend loop and connects SSE
		nil,   // no onProjectReady callback in client mode
	)

	// Notify daemon to stop when the TUI client exits
	quitBody, _ := json.Marshal(map[string]string{"key": "quit"})
	req, err := http.NewRequest("POST", baseURL+"/tinywasm/action", bytes.NewReader(quitBody))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer " + apiKey)
		}
		resp, err := http.DefaultClient.Do(req) // best-effort, ignore error
		if err == nil {
			resp.Body.Close()
		}
	}
}
