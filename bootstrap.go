package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/tinywasm/devflow"
	"github.com/tinywasm/mcpserve"
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
	McpToolHandlers []mcpserve.ToolProvider
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

func runDaemon(cfg BootstrapConfig) {
	// Headless mode
	// Ensure we have a valid Logger for daemon itself
	logger := cfg.Logger
	if logger == nil {
		logger = func(messages ...any) { fmt.Println(messages...) }
	}

	logger("Starting TinyWASM Global MCP Daemon on port 3030...")

	mcpPort := "3030"
	if p := os.Getenv("TINYWASM_MCP_PORT"); p != "" {
		mcpPort = p
	}

	mcpConfig := mcpserve.Config{
		Port:          mcpPort,
		ServerName:    "TinyWasm - Global MCP Server",
		ServerVersion: "1.0.0",
		AppName:       "tinywasm",
		AppVersion:    cfg.Version,
	}

	// Create an empty TUI stub for the daemon if not provided
	var ui TuiInterface
	exitChan := make(chan bool)
	if cfg.TuiFactory != nil {
		ui = cfg.TuiFactory(exitChan, false, "") // daemon has no TUI client, no SSE needed
	} else {
		ui = NewHeadlessTUI(logger)
	}

	// We will create the MCP handler globally first but without tools so we can pass it
	// to the daemon tool provider.
	mcpHandler := mcpserve.NewHandler(mcpConfig, nil, ui, exitChan)
	mcpHandler.SetLog(logger)
	mcpHandler.ConfigureIDEs()

	// Define the daemon tool provider that controls the project lifecycles
	dtp := newDaemonToolProvider(cfg, mcpHandler, logger)

	// Since NewHandler accepts full slice, we can add it manually or recreate the handler.
	// We'll just recreate it for simplicity, since it's cleaner:
	mcpHandler = mcpserve.NewHandler(mcpConfig, append(cfg.McpToolHandlers, dtp), ui, exitChan)
	mcpHandler.SetLog(logger)
	mcpHandler.ConfigureIDEs()
	// CRITICAL: dtp was created with the first mcpHandler (no sseHub yet).
	// Update it to point to this final handler so RelayLog publishes to the real SSE hub.
	dtp.mcpHandler = mcpHandler

	// Handle UI Webhooks (e.g. from the TUI Client when user presses "q" or "r")
	mcpHandler.OnUIAction(func(key string) {
		switch key {
		case "q":
			logger("Stop command received from UI")
			dtp.stopProject()
			// We intentionally don't close exitChan here to keep the Daemon alive,
			// just the project dies.
		case "r":
			// A true hot reload could be triggered by touching a watched file or signaling the watcher
			// For now, restarting the project is the closest equivalent
			logger("Restart command received from UI")
			dtp.restartCurrentProject()
		default:
			logger("Unknown UI action:", key)
		}
	})

	// Optional: auto-start the project in the current directory
	// so `tinywasm -mcp` behaves similarly to before for the local folder.
	if cfg.StartDir != "" && cfg.StartDir != "/" {
		go func() {
			// Give the server a moment to start
			time.Sleep(500 * time.Millisecond)
			dtp.startProject(cfg.StartDir)
		}()
	}

	// Block forever serving MCP and SSE
	mcpHandler.Serve()
}

// daemonToolProvider implements mcpserve.ToolProvider to expose global daemon tools
// and manages the lifecycle of the running project instance.
type daemonToolProvider struct {
	cfg           BootstrapConfig
	mcpHandler    *mcpserve.Handler
	logger        func(messages ...any)
	projectCancel context.CancelFunc
	projectDone   chan struct{}
	mu            sync.Mutex
	lastPath      string // Keep track of the last path for remote restarts
}

func newDaemonToolProvider(cfg BootstrapConfig, mcp *mcpserve.Handler, logger func(messages ...any)) *daemonToolProvider {
	return &daemonToolProvider{
		cfg:        cfg,
		mcpHandler: mcp,
		logger:     logger,
	}
}

func (d *daemonToolProvider) GetMCPToolsMetadata() []mcpserve.ToolMetadata {
	return []mcpserve.ToolMetadata{
		{
			Name:        "start_development",
			Description: "Start a TinyWASM project environment (Server, WASM compiler, Assets, Browser) in the specified directory. Will stop any currently running project first.",
			Parameters: []mcpserve.ParameterMetadata{
				{
					Name:        "ide_name",
					Description: "Name of the IDE or LLM client making the request (e.g., 'vsc', 'cursor')",
					Required:    true,
					Type:        "string",
				},
				{
					Name:        "project_path",
					Description: "Absolute or relative path to the TinyWASM project directory to start",
					Required:    true,
					Type:        "string",
				},
			},
			Execute: func(args map[string]any) {
				path, ok := args["project_path"].(string)
				if !ok {
					d.logger("Error: project_path is required and must be a string")
					return
				}

				ide, ok := args["ide_name"].(string)
				if !ok {
					ide = "unknown"
				}

				d.logger("Starting development environment for:", path, "(requested by:", ide, ")")
				d.startProject(path)
			},
		},
	}
}

func (d *daemonToolProvider) stopProject() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.projectCancel != nil {
		d.projectCancel()
		d.projectCancel = nil
	}
}

func (d *daemonToolProvider) restartCurrentProject() {
	d.mu.Lock()
	path := d.lastPath
	d.mu.Unlock()

	if path != "" {
		d.startProject(path)
	} else {
		d.logger("Cannot restart: no project has been started yet.")
	}
}

func (d *daemonToolProvider) startProject(projectPath string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.lastPath = projectPath

	// 1. Cancel previous project
	if d.projectCancel != nil {
		d.projectCancel()
	}

	// 2. Block until port 8080 unbinds
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

portLoop:
	for {
		select {
		case <-timeout:
			d.logger("Warning: Port 8080 still active after timeout, attempting to start anyway...")
			break portLoop
		case <-ticker.C:
			conn, err := net.Dial("tcp", "localhost:"+os.Getenv("PORT"))
			if err != nil {
				// We assume it's 8080 if PORT isn't set, this dial check uses the same env as Start
				// Actually, Start sets a default 8080 if empty, but we can't easily know here.
				// For safety, let's just dial 8080 as a heuristic.
				conn8080, err8080 := net.Dial("tcp", "localhost:8080")
				if err8080 != nil {
					break portLoop
				}
				conn8080.Close()
			} else {
				conn.Close()
			}
		}
	}

	d.logger("Project Restart logic: starting new project at", projectPath)

	ctx, cancel := context.WithCancel(context.Background())
	d.projectCancel = cancel
	d.projectDone = make(chan struct{})

	go func() {
		defer close(d.projectDone)
		d.runProjectLoop(ctx, projectPath)
	}()
}

func (d *daemonToolProvider) runProjectLoop(ctx context.Context, projectPath string) {
	// Create a separate run channel for this project
	runExitChan := make(chan bool)
	headlessTui := NewHeadlessTUI(d.logger)
	// Wire component loggers to the daemon SSE hub so the client TUI receives structured logs
	headlessTui.RelayLog = func(tabTitle, handlerName, color, msg string) {
		d.mcpHandler.PublishTabLog(tabTitle, handlerName, color, msg)
	}
	browser := d.cfg.BrowserFactory(headlessTui, runExitChan)

	// We wire context cancellation to the channels
	go func() {
		select {
		case <-ctx.Done():
			d.logger("Context cancelled, stopping project loop...")
			close(runExitChan)
		case <-runExitChan:
			// app stopped itself
		}
	}()

	// Let's start the app loop here
	for {
		restart := Start(
			projectPath,
			d.logger,
			headlessTui,
			browser,
			d.cfg.DB,
			runExitChan,
			d.cfg.ServerFactory,
			d.cfg.GitHubAuth,
			d.cfg.GitHandler,
			d.cfg.GoModHandler,
			true,  // headless
			false, // clientMode
			// Empty tools
		)
		if !restart || ctx.Err() != nil {
			break
		}
		d.logger("Restarting project loop...")
		// Recreate exit channel for next loop
		runExitChan = make(chan bool)
		go func(c chan bool) {
			select {
			case <-ctx.Done():
				close(c)
			case <-c:
			}
		}(runExitChan)
	}
	d.logger("Project loop ended for", projectPath)
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
	)
}

func isPortOpen(port string) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", port), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		conn.Close()
		return true
	}
	return false
}

// isDaemonVersionCurrent fetches /version from the running daemon and compares to ours.
// Returns true if versions match (daemon is current), false if stale or unreachable.
func isDaemonVersionCurrent(port, version string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:" + port + "/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var data struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false
	}
	return data.Version == version
}

// killDaemon sends SIGTERM to any running tinywasm daemon process.
func killDaemon() {
	exec.Command("pkill", "-f", "tinywasm -mcp").Run()
}

// waitForPortFree polls until the port is no longer listening (max 5s).
func waitForPortFree(port string) {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			return
		case <-ticker.C:
			if !isPortOpen(port) {
				return
			}
		}
	}
}

// waitForPortReady polls until the port is accepting connections (max 5s).
func waitForPortReady(port string) {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			fmt.Println("Timeout waiting for daemon to start")
			os.Exit(1)
		case <-ticker.C:
			if isPortOpen(port) {
				return
			}
		}
	}
}

func startDaemonProcess(dir string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Start detached process
	cmd := exec.Command(exe, "-mcp")
	cmd.Dir = dir

	logPath := filepath.Join(dir, "tinywasm-daemon.log")
	logFile, err := os.Create(logPath)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	return cmd.Start()
}
