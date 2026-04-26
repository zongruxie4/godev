package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	twctx "github.com/tinywasm/context"
	"github.com/tinywasm/devflow"
	twfmt "github.com/tinywasm/fmt"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/sse"
)

// TestMode disables browser auto-start when running tests
var TestMode bool

// Start is called from main.go with UI, Browser and DB passed as parameters
// CRITICAL: UI, Browser and DB instances created in main.go, passed here as interfaces
// mcpToolHandlers: optional external Handlers that implement Tools() for MCP tool discovery
// onProjectReady: optional callback called after handler initialization (for daemon mode to set up proxy)
func Start(startDir string, logger any, ui TuiInterface, browser BrowserInterface, db DB, ExitChan chan bool, serverFactory ServerFactory, githubAuth any, gitHandler devflow.GitClient, goModHandler devflow.GoModInterface, headless bool, clientMode bool, onProjectReady func(*Handler), mcpToolHandlers ...mcp.ToolProvider) bool {

	var loggerFunc func(messages ...any)
	if l, ok := logger.(func(...any)); ok {
		loggerFunc = l
	} else if l, ok := logger.(*Logger); ok {
		loggerFunc = l.Logger
	}

	// Initialize Go Handler
	GoHandler, _ := devflow.NewGo(gitHandler)
	if GoHandler != nil {
		GoHandler.SetRootDir(startDir)
	}
	if goModHandler != nil {
		goModHandler.SetRootDir(startDir)
	}

	h := &Handler{
		FrameworkName: "TINYWASM",
		RootDir:       startDir,
		Tui:           ui, // UI passed from main.go
		ExitChan:      ExitChan,
		Logger:        loggerFunc,

		DB:            db,
		serverFactory: serverFactory,
		GitHandler:    gitHandler,
		GoHandler:     GoHandler,
		Browser:       browser,
		GitHubAuth:    githubAuth,
		GoModHandler:  goModHandler,
	}

	// Noop initial logger to avoid nil check issues
	h.Logger = loggerFunc

	// Check if we are in dev mode
	h.CheckDevMode()

	// Wire gitignore notification
	if !TestMode && gitHandler != nil {
		gitHandler.SetShouldWrite(h.IsInitializedProject)
	}

	// Validate directory
	homeDir, _ := os.UserHomeDir()
	if startDir == homeDir || startDir == "/" {
		loggerFunc("Cannot run tinywasm in user root directory. Please run in a Go project directory")
		return false
	}

	// Secondary guard: reject if startDir is inside a project but not its root
	if root, err := devflow.FindProjectRoot(startDir); err == nil && root != startDir {
		loggerFunc(twfmt.Translate("Directory", "Not", "Initialized").String())
		return false
	}

	var wg sync.WaitGroup
	wg.Add(1) // UI goroutine; MCP added below only in standalone (non-headless) mode

	// ADD SECTIONS using the passed UI interface
	// CRITICAL: Initialize sections BEFORE starting lifecycle
	h.SectionBuild = h.AddSectionBUILD()
	h.AddSectionDEPLOY()
	h.SectionMCP = h.AddSectionMCP()

	// Register GitHubAuth in TUI FIRST (so it gets the TUI logger)
	// This must happen BEFORE starting the auth Future
	if h.GitHubAuth != nil {
		h.Tui.AddHandler(h.GitHubAuth, "#6e40c9", h.SectionBuild) // Purple for GitHub
	}

	// Early return for clientMode: we just needed to construct the TUI sections.
	// We don't want to run Github auth, GoNew, watchers, or the local MCP server in the client.
	if clientMode {
		var clientWg sync.WaitGroup
		clientWg.Add(1)
		go h.Tui.Start(&clientWg)
		clientWg.Wait()
		return false
	}

	// NOW start GitHub auth in background (after TUI registration)
	// This ensures the TUI logger is set before auth messages are sent
	githubFuture := devflow.NewFuture(func() (any, error) {
		if githubAuth != nil {
			if auth, ok := githubAuth.(devflow.GitHubAuthenticator); ok {
				return devflow.NewGitHub(loggerFunc, auth)
			}
		}
		return devflow.NewGitHub(loggerFunc)
	})

	// Initialize GoNew orchestrator with the future
	GoNew := devflow.NewGoNew(gitHandler, githubFuture, GoHandler)
	GoNew.SetLog(loggerFunc)
	h.GoNew = GoNew
	// Prevents goroutine leak: drain future when app exits so the
	// internal goroutine can proceed to close(f.done) and exit.
	go func() {
		select {
		case <-ExitChan:
			<-githubFuture.Ready() // unblock the future's sender goroutine
		case <-githubFuture.Ready():
			// already resolved, nothing to do
		}
	}()

	if !h.IsPartOfProject() {
		sectionWizard := h.AddSectionWIZARD(func() {
			h.OnProjectReady(&wg)
			// Call onProjectReady callback (used by daemon to set up tool proxy)
			if onProjectReady != nil {
				onProjectReady(h)
			}
		})
		h.Tui.SetActiveTab(sectionWizard)
	} else {
		h.OnProjectReady(&wg)
		// Call onProjectReady callback (used by daemon to set up tool proxy)
		if onProjectReady != nil {
			onProjectReady(h)
		}
	}

	if !headless {
		// Standalone mode: create and serve the project-level HTTP server on port 3030.
		// In headless/daemon-sub-project mode this is skipped to avoid a port conflict
		// with the already-running global daemon MCP.
		mcpPort := "3030"
		if p := os.Getenv("TINYWASM_MCP_PORT"); p != "" {
			mcpPort = p
		}

		// Create SSE server for log transport
		tinySSE := sse.New(&sse.Config{})
		sseServer := tinySSE.Server(&sse.ServerConfig{
			ChannelProvider:     &logChannelProvider{},
			ClientChannelBuffer: 256,
			HistoryReplayBuffer: 100,
			ReplayAllOnConnect:  true,
		})

		ssePub := NewSSEPublisher(sseServer)

		// Wire Logger redirection to SSE
		if l, ok := logger.(*Logger); ok {
			l.Redir = func(messages ...any) {
				ssePub.PublishLog(l.sprint(messages...))
			}
		}

		h.Logger = func(messages ...any) {
			loggerFunc(messages...)
			if l, ok := logger.(*Logger); ok {
				ssePub.PublishLog(l.sprint(messages...))
			} else {
				// Fallback if not using our Logger struct
				ssePub.PublishLog(fmt.Sprint(messages...))
			}
		}

		appVersion := "1.0.0"
		// Try to get version from DB if available, or just use 1.0.0
		if val, err := h.DB.Get("TINYWASM_VERSION"); err == nil && val != "" {
			appVersion = val
		}

		mcpConfig := mcp.Config{
			Name:    "TinyWasm - Full-stack Go+WASM Dev Environment",
			Version: appVersion,
			Auth:    mcp.OpenAuthorizer(),
			SSE:     sseServer,
		}

		// Use buildProjectProviders for single source of truth
		toolHandlers := buildProjectProviders(h)
		toolHandlers = append(toolHandlers, mcpToolHandlers...)

		var err error
		h.MCP, err = mcp.NewServer(mcpConfig, toolHandlers)
		if err != nil {
			loggerFunc("Failed to initialize MCP Server:", err)
			return false
		}

		// Configure IDEs (standalone mode)
		if err := ConfigureIDEs("tinywasm", appVersion, mcpPort, ""); err != nil {
			loggerFunc("Warning: Failed to configure IDEs:", err)
		}

		h.Tui.AddHandler(h.MCP, colorOrangeLight, h.SectionMCP)
		SetActiveHandler(h)

		mux := http.NewServeMux()
		mux.Handle("/logs", sseServer)
		mux.HandleFunc("POST /mcp", func(w http.ResponseWriter, r *http.Request) {
			var msg []byte
			if r.Body != nil {
				msg, _ = io.ReadAll(r.Body)
			}
			ctx := twctx.Background()
			resp := h.MCP.HandleMessage(ctx, msg)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})
		mux.HandleFunc("GET /tinywasm/state", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(h.Tui.GetHandlerStates())
		})
		mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(appVersion))
		})

		server := &http.Server{
			Addr:    ":" + mcpPort,
			Handler: mux,
		}

		// Start HTTP server
		wg.Add(1)
		go func() {
			defer wg.Done()
			go func() {
				<-ExitChan
				server.Close()
			}()
			loggerFunc("Standalone listener on", server.Addr)
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				loggerFunc("Standalone server error:", err)
			}
		}()

		// Start the UI
		go h.Tui.Start(&wg, ExitChan)
	} else {
		// Headless mode: no HTTP, no UI. Keep alive until ExitChan is closed.
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ExitChan
		}()
		wg.Done() // satisfy the initial wg.Add(1) for UI
	}

	wg.Wait()
	return h.RestartRequested
}
