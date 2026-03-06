package app

import (
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
	Logger          func(messages ...any)
	DB              DB
	GitHandler      devflow.GitClient
	GoModHandler    devflow.GoModInterface
	ServerFactory   ServerFactory
	TuiFactory      func(exitChan chan bool, clientMode bool, clientURL string) TuiInterface
	BrowserFactory  func(ui TuiInterface, exitChan chan bool) BrowserInterface
	GitHubAuth      any
	McpToolHandlers []mcp.ToolProvider
}

// Bootstrap is the main entry point for the application logic
func Bootstrap(cfg BootstrapConfig) {
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
			if err := startDaemonProcess(cfg.StartDir); err != nil {
				fmt.Printf("Failed to restart daemon: %v\n", err)
				os.Exit(1)
			}
			waitForPortReady("3030")
		}
		runClient(cfg)
	} else {
		// Port free -> Start Daemon in background, then run Client
		if err := startDaemonProcess(cfg.StartDir); err != nil {
			fmt.Printf("Failed to start daemon: %v\n", err)
			os.Exit(1)
		}
		waitForPortReady("3030")
		runClient(cfg)
	}
}

// logChannelProvider implements sse.ChannelProvider
type logChannelProvider struct{}

func (p *logChannelProvider) ResolveChannels(r *http.Request) ([]string, error) {
	return []string{"logs"}, nil
}


func runClient(cfg BootstrapConfig) {
	// Client mode
	// Use real TUI with SSE client enabled to receive logs from the daemon
	exitChan := make(chan bool)
	mcpPort := "3030"
	if p := os.Getenv("TINYWASM_MCP_PORT"); p != "" {
		mcpPort = p
	}
	clientURL := "http://localhost:" + mcpPort + "/logs"
	ui := cfg.TuiFactory(exitChan, true, clientURL)

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
}

