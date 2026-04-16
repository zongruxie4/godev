package app

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	twctx "github.com/tinywasm/context"
	twfmt "github.com/tinywasm/fmt"
	twjson "github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/sse"
	"strings"
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

	// Create an empty TUI stub for the daemon if not provided
	var ui TuiInterface
	exitChan := make(chan bool)
	if cfg.TuiFactory != nil {
		ui = cfg.TuiFactory(exitChan, false, "", "") // daemon has no TUI client, no SSE needed
	} else {
		ui = NewHeadlessTUI(logger)
	}

	// Create SSE server (tinywasm/sse)
	tinySSE := sse.New(&sse.Config{})
	sseServer := tinySSE.Server(&sse.ServerConfig{
		ChannelProvider:     &logChannelProvider{},
		ClientChannelBuffer: 256,
		HistoryReplayBuffer: 100,
		ReplayAllOnConnect:  true,
	})

	// Load or create API key for this daemon instance
	apiKey, err := loadOrCreateAPIKey(cfg.APIKeyPath)
	if err != nil {
		fmt.Printf("Failed to generate API key: %v\n", err)
		os.Exit(1)
	}

	var auth mcp.Authorizer
	if apiKey != "" {
		auth = mcp.NewTokenAuthorizer(apiKey)
	} else {
		auth = mcp.OpenAuthorizer()
	}

	mcpConfig := mcp.Config{
		Name:    "TinyWasm - Global MCP Server",
		Version: cfg.Version,
		Auth:    auth,
		SSE:     sseServer,
	}

	// proxy is a FIXED provider
	proxy := NewProjectToolProxy()

	// Define the daemon tool provider that controls the project lifecycles
	dtp := newDaemonToolProvider(cfg, logger)

	toolProviders := append(cfg.McpToolHandlers, dtp, proxy)
	mcpServer, err := mcp.NewServer(mcpConfig, toolProviders)
	if err != nil {
		fmt.Printf("Failed to initialize MCP Server: %v\n", err)
		os.Exit(1)
	}

	// Configure IDEs
	if err := ConfigureIDEs("tinywasm", cfg.Version, mcpPort, apiKey); err != nil {
		logger("Warning: Failed to configure IDEs:", err)
	}

	ssePub := NewSSEPublisher(sseServer)

	// Update dtp to store proxy, mcpServer and ssePub
	dtp.toolProxy = proxy
	dtp.mcpServer = mcpServer
	dtp.ssePub = ssePub

	mux := http.NewServeMux()

	// SSE endpoint (from tinywasm/sse)
	mux.Handle("/logs", sseServer)

	// Helper for common authorization and context creation
	getAuthCtx := func(r *http.Request) (*twctx.Context, string) {
		ctx := twctx.Background()
		authHeader := r.Header.Get("Authorization")
		token := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
		ctx.Set(mcp.CtxKeyAuthToken, token)
		return ctx, token
	}

	// MCP JSON-RPC endpoint
	mux.HandleFunc("POST /mcp", func(w http.ResponseWriter, r *http.Request) {
		var msg []byte
		if r.Body != nil {
			msg, _ = io.ReadAll(r.Body)
		}

		// Extract values from JSON-RPC body to intercept custom app methods.
		// Use ExtractJSONValue to avoid encoding/json and handle int/string IDs.
		method := string(unquote(mcp.ExtractJSONValue(msg, "method")))
		id := string(mcp.ExtractJSONValue(msg, "id"))
		if id == "" {
			id = "null"
		}
		params := mcp.ExtractJSONValue(msg, "params")

		switch method {
		case "tinywasm/state":
			_, token := getAuthCtx(r)
			if _, err := auth.Authorize(token); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"Unauthorized"}}`, id)))
				return
			}

			dtp.mu.Lock()
			projectTui := dtp.projectTui
			dtp.mu.Unlock()

			var stateJSON []byte
			if projectTui != nil {
				stateJSON = projectTui.GetHandlerStates()
			} else {
				stateJSON = ui.GetHandlerStates()
			}

			// result must be a JSON-encoded string (double-encoded) per mcp wire protocol
			resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%q}`,
				id, string(stateJSON))
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(resp))

		case "tinywasm/action":
			_, token := getAuthCtx(r)
			if _, err := auth.Authorize(token); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"Unauthorized"}}`, id)))
				return
			}

			// params can be double-encoded string or direct object
			pBytes := params
			if len(params) > 0 && params[0] == '"' {
				pBytes = unquote(params)
			}

			key := string(unquote(mcp.ExtractJSONValue(pBytes, "key")))
			value := string(unquote(mcp.ExtractJSONValue(pBytes, "value")))

			handled := false
			dtp.mu.Lock()
			projectTui := dtp.projectTui
			dtp.mu.Unlock()
			if projectTui != nil && projectTui.DispatchAction(key, value) {
				handled = true
			} else if ui.DispatchAction(key, value) {
				handled = true
			}
			if !handled {
				switch key {
				case "start":
					if value != "" {
						logger("Start command received for path:", value)
						go dtp.startProject(value)
					}
				case "stop":
					dtp.stopProject()
				case "restart":
					dtp.restartCurrentProject()
				default:
					logger("Unknown UI action:", key)
				}
			}

			resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":"OK"}`, id)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(resp))

		default:
			// Standard MCP protocol
			ctx, _ := getAuthCtx(r)
			resp := mcpServer.HandleMessage(ctx, msg)
			w.Header().Set("Content-Type", "application/json")
			var out []byte
			if f, ok := resp.(twfmt.Fielder); ok {
				twjson.Encode(f, &out)
			}
			if len(out) == 0 {
				// Fallback if not a Fielder or empty
				w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal Error"}}`))
			} else {
				w.Write(out)
			}
		}
	})

	// Register action dispatcher
	mux.HandleFunc("POST /tinywasm/action", func(w http.ResponseWriter, r *http.Request) {
		_, token := getAuthCtx(r)
		if _, err := auth.Authorize(token); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		body, _ := io.ReadAll(r.Body)
		key := string(unquote(mcp.ExtractJSONValue(body, "key")))
		value := string(unquote(mcp.ExtractJSONValue(body, "value")))

		dtp.mu.Lock()
		projectTui := dtp.projectTui
		dtp.mu.Unlock()

		handled := false
		if projectTui != nil && projectTui.DispatchAction(key, value) {
			handled = true
		} else if ui.DispatchAction(key, value) {
			handled = true
		}

		if !handled {
			switch key {
			case "start":
				if value != "" {
					logger("Start command received for path:", value)
					go dtp.startProject(value)
				}
			case "stop":
				logger("Stop command received from UI")
				dtp.stopProject()
			case "restart":
				logger("Restart command received from UI")
				dtp.restartCurrentProject()
			default:
				logger("Unknown UI action:", key)
			}
		}
		w.Write([]byte("OK"))
	})

	// Register state provider
	mux.HandleFunc("GET /tinywasm/state", func(w http.ResponseWriter, r *http.Request) {
		_, token := getAuthCtx(r)
		if _, err := auth.Authorize(token); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		dtp.mu.Lock()
		projectTui := dtp.projectTui
		dtp.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if projectTui != nil {
			w.Write(projectTui.GetHandlerStates())
		} else {
			w.Write(ui.GetHandlerStates())
		}
	})

	// Server version endpoint
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(cfg.Version))
	})

	server := &http.Server{
		Addr:    ":" + mcpPort,
		Handler: mux,
	}

	go func() {
		<-exitChan
		server.Close()
	}()

	logger("Daemon listening on", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger("Server error:", err)
	}
}

// daemonToolProvider implements mcp.ToolProvider to expose global daemon tools
// and manages the lifecycle of the running project instance.
type daemonToolProvider struct {
	cfg           BootstrapConfig
	mcpServer     *mcp.Server
	ssePub        *SSEPublisher
	toolProxy     *ProjectToolProxy // Updated when projects start/stop
	logger        func(messages ...any)
	projectCancel chan bool
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

func (d *daemonToolProvider) Tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "start_development",
			Description: "Start a TinyWASM project environment (Server, WASM compiler, Assets, Browser) in the specified directory. Will stop any currently running project first.",
			InputSchema: `{
				"type": "object",
				"properties": {
					"ide_name": { "type": "string", "description": "Name of the IDE or LLM client making the request" },
					"project_path": { "type": "string", "description": "Absolute or relative path to the TinyWASM project directory" }
				},
				"required": ["project_path"]
			}`,
			Resource: "project",
			Action:   'c',
			Execute: func(ctx *twctx.Context, req mcp.Request) (*mcp.Result, error) {
				argsBytes := []byte(req.Params.Arguments)
				ideName := string(unquote(mcp.ExtractJSONValue(argsBytes, "ide_name")))
				projectPath := string(unquote(mcp.ExtractJSONValue(argsBytes, "project_path")))

				if projectPath == "" {
					return nil, fmt.Errorf("project_path is required")
				}

				d.logger("Starting development environment for:", projectPath, "(requested by:", ideName, ")")
				d.startProject(projectPath)
				return mcp.Text("Development environment starting..."), nil
			},
		},
	}
}

func unquote(b []byte) []byte {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return b
	}
	// Single-pass unquoting to avoid sequential replacement bugs (e.g., \\n)
	var res strings.Builder
	res.Grow(len(b) - 2)
	s := string(b[1 : len(b)-1])
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '\\':
				res.WriteByte('\\')
			case '"':
				res.WriteByte('"')
			case 'n':
				res.WriteByte('\n')
			case 'r':
				res.WriteByte('\r')
			case 't':
				res.WriteByte('\t')
			default:
				res.WriteByte(s[i+1])
			}
			i++
		} else {
			res.WriteByte(s[i])
		}
	}
	return []byte(res.String())
}

func (d *daemonToolProvider) stopProject() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.projectCancel != nil {
		close(d.projectCancel)
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
		close(d.projectCancel)
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

	ctx := twctx.Background()
	cancel := make(chan bool)
	d.projectCancel = cancel
	d.projectDone = make(chan struct{})

	go func() {
		defer close(d.projectDone)
		d.runProjectLoop(ctx, projectPath, cancel)
	}()
}

func (d *daemonToolProvider) runProjectLoop(ctx *twctx.Context, projectPath string, cancel chan bool) {
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

	// We wire cancellation to the channels
	go func() {
		select {
		case <-cancel:
			d.logger("Stop signaled, stopping project loop...")
			close(runExitChan)
		case <-runExitChan:
			// project stopped itself
		}
	}()

	// defer cleanup: clear proxy
	defer func() {
		d.toolProxy.SetActive()
		d.logger("Project loop cleanup: proxy cleared")
	}()

	// Let's start the app loop here
	for {
		// Callback to set up proxy after handler is initialized
		onProjectReady := func(h *Handler) {
			providers := buildProjectProviders(h)
			d.toolProxy.SetActive(providers...)
			d.logger("ProjectToolProxy activated:", len(providers), "providers")
			d.ssePub.PublishStateRefresh()
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

		select {
		case <-cancel:
			restart = false
		default:
		}

		if !restart {
			break
		}
		d.logger("Restarting project loop...")
		// Recreate exit channel for next loop
		runExitChan = make(chan bool)
		go func(c chan bool, cl chan bool) {
			select {
			case <-cl:
				close(c)
			case <-c:
			}
		}(runExitChan, cancel)
	}
	d.logger("Project loop ended for", projectPath)
}
