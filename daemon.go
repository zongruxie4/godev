package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/tinywasm/mcp"
	"github.com/tinywasm/sse"
)

// runDaemon starts the global MCP daemon on port 3030
func runDaemon(cfg BootstrapConfig) {
	logger := cfg.Logger
	if logger == nil {
		logger = func(messages ...any) {}
	}

	logger("Starting TinyWASM Global MCP Daemon on port 3030...")

	mcpPort := "3030"
	if p := os.Getenv("TINYWASM_MCP_PORT"); p != "" {
		mcpPort = p
	}

	mcpConfig := mcp.Config{
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
		ui = cfg.TuiFactory(exitChan, false, "", "") // daemon has no TUI client, no SSE needed
	} else {
		ui = NewHeadlessTUI(logger)
	}

	// Create SSE server (tinywasm/sse) and inject into Handler (mcp.SSEHub interface)
	tinySSE := sse.New(&sse.Config{})
	sseHub := &sseHubAdapter{tinySSE.Server(&sse.ServerConfig{
		ChannelProvider:     &logChannelProvider{},
		ClientChannelBuffer: 256,
		HistoryReplayBuffer: 100,
		ReplayAllOnConnect:  true,
	})}

	// Load or create API key for this daemon instance
	apiKey, err := loadOrCreateAPIKey(cfg.APIKeyPath)
	if err != nil {
		fmt.Printf("Failed to generate API key: %v\n", err)
		os.Exit(1)
	}

	// FIX: create proxy BEFORE mcpHandler (root bug — was created after)
	proxy := NewProjectToolProxy()

	// Define the daemon tool provider that controls the project lifecycles
	dtp := newDaemonToolProvider(cfg, logger)

	// FIX: proxy is a FIXED provider — MCPServer rebuilds include it
	toolProviders := append(cfg.McpToolHandlers, dtp, proxy)
	mcpHandler := mcp.NewHandler(mcpConfig, sseHub, toolProviders)
	mcpHandler.SetLog(logger)

	// Wire auth: ALWAYS explicit — mcp.Handler denies all by default.
	// Open mode (no APIKeyPath) is a conscious opt-in, not a silent fallback.
	if apiKey != "" {
		mcpHandler.SetAuth(mcp.NewTokenAuthorizer(apiKey))
		mcpHandler.SetAPIKey(apiKey) // written into IDE config Authorization headers
	} else {
		mcpHandler.SetAuth(mcp.OpenAuthorizer()) // explicit opt-in: local/trusted environment
	}
	mcpHandler.ConfigureIDEs()

	ssePub := NewSSEPublisher(sseHub)

	// Update dtp to store proxy, mcpHandler and ssePub
	dtp.toolProxy = proxy
	dtp.mcpHandler = mcpHandler
	dtp.ssePub = ssePub

	// Method names defined here in app — mcp.Handler is agnostic.
	const (
		methodAction = "tinywasm/action"
		methodState  = "tinywasm/state"
	)

	// Register action dispatcher — app owns the method name and param schema.
	mcpHandler.RegisterMethod(methodAction, func(ctx context.Context, params []byte) (any, error) {
		var p struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		json.Unmarshal(params, &p)

		dtp.mu.Lock()
		projectTui := dtp.projectTui
		dtp.mu.Unlock()
		if projectTui != nil && projectTui.DispatchAction(p.Key, p.Value) {
			return "OK", nil
		}
		if ui.DispatchAction(p.Key, p.Value) {
			return "OK", nil
		}
		switch p.Key {
		case "start":
			if p.Value != "" {
				logger("Start command received for path:", p.Value)
				go dtp.startProject(p.Value)
			}
		case "stop":
			logger("Stop command received from UI")
			dtp.stopProject()
		case "restart":
			logger("Restart command received from UI")
			dtp.restartCurrentProject()
		default:
			logger("Unknown UI action:", p.Key)
		}
		return "OK", nil
	})

	// Register state provider — app owns the method name and return schema.
	mcpHandler.RegisterMethod(methodState, func(ctx context.Context, _ []byte) (any, error) {
		dtp.mu.Lock()
		projectTui := dtp.projectTui
		dtp.mu.Unlock()
		if projectTui != nil {
			return json.RawMessage(projectTui.GetHandlerStates()), nil
		}
		return json.RawMessage(ui.GetHandlerStates()), nil
	})

	// Block forever serving MCP (which includes /mcp, /logs, /action, /state, /version)
	mcpHandler.Serve(exitChan)
}

// daemonToolProvider implements mcp.ToolProvider to expose global daemon tools
// and manages the lifecycle of the running project instance.
type daemonToolProvider struct {
	cfg           BootstrapConfig
	mcpHandler    *mcp.Handler
	ssePub        *SSEPublisher
	toolProxy     *ProjectToolProxy // Updated when projects start/stop
	logger        func(messages ...any)
	projectCancel context.CancelFunc
	projectDone   chan struct{}
	projectTui    *HeadlessTUI // Current project's TUI (updated per project start)
	mu            sync.Mutex
	lastPath      string // Keep track of the last path for remote restarts
}

func newDaemonToolProvider(cfg BootstrapConfig, logger func(messages ...any)) *daemonToolProvider {
	return &daemonToolProvider{
		cfg:    cfg,
		logger: logger,
	}
}

func (d *daemonToolProvider) GetMCPTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "start_development",
			Description: "Start a TinyWASM project environment (Server, WASM compiler, Assets, Browser) in the specified directory. Will stop any currently running project first.",
			Parameters: []mcp.Parameter{
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
		d.ssePub.PublishTabLog(tabTitle, handlerName, color, msg)
	}

	// Register project TUI so /state and /action can reach project handlers
	d.mu.Lock()
	d.projectTui = headlessTui
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		d.projectTui = nil
		d.mu.Unlock()
	}()
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

	// defer cleanup: clear proxy then trigger rebuild
	defer func() {
		d.toolProxy.SetActive()
		d.mcpHandler.SetDynamicProviders() // rebuild with empty proxy
		d.logger("Project loop cleanup: proxy cleared")
	}()

	// Let's start the app loop here
	for {
		// Callback to set up proxy after handler is initialized
		onProjectReady := func(h *Handler) {
			providers := buildProjectProviders(h)
			d.toolProxy.SetActive(providers...)
			// Pass providers directly so rebuildMCPServer can resolve Loggable
			// per individual provider (proxy indirection breaks Loggable type assertion).
			d.mcpHandler.SetDynamicProviders(providers...)
			d.logger("ProjectToolProxy activated:", len(providers), "providers")
			d.ssePub.PublishStateRefresh() // signal only — devtui re-fetches via JSON-RPC
		}

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
			onProjectReady,
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
