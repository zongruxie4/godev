package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/tinywasm/devflow"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/sse"
)

// TestMode disables browser auto-start when running tests
var TestMode bool

// Start is called from main.go with UI, Browser and DB passed as parameters
// CRITICAL: UI, Browser and DB instances created in main.go, passed here as interfaces
// mcpToolHandlers: optional external Handlers that implement GetMCPTools() for MCP tool discovery
// onProjectReady: optional callback called after handler initialization (for daemon mode to set up proxy)
func Start(startDir string, logger func(messages ...any), ui TuiInterface, browser BrowserInterface, db DB, ExitChan chan bool, serverFactory ServerFactory, githubAuth any, gitHandler devflow.GitClient, goModHandler devflow.GoModInterface, headless bool, clientMode bool, onProjectReady func(*Handler), mcpToolHandlers ...mcp.ToolProvider) bool {

	// Initialize Go Handler
	GoHandler, _ := devflow.NewGo(gitHandler)
	GoHandler.SetRootDir(startDir)

	h := &Handler{
		FrameworkName: "TINYWASM",
		RootDir:       startDir,
		Tui:           ui, // UI passed from main.go
		ExitChan:      ExitChan,
		Logger:        logger,

		DB:            db,
		serverFactory: serverFactory,
		GitHandler:    gitHandler,
		GoHandler:     GoHandler,
		Browser:       browser,
		GitHubAuth:    githubAuth,
		GoModHandler:  goModHandler,
	}

	// Noop initial logger to avoid nil check issues
	h.Logger = logger

	// Check if we are in dev mode
	h.CheckDevMode()

	// Wire gitignore notification
	if !TestMode {
		gitHandler.SetShouldWrite(h.IsInitializedProject)
	}

	// Validate directory
	homeDir, _ := os.UserHomeDir()
	if startDir == homeDir || startDir == "/" {
		logger("Cannot run tinywasm in user root directory. Please run in a Go project directory")
		return false
	}

	var wg sync.WaitGroup
	wg.Add(1) // UI goroutine; MCP added below only in standalone (non-headless) mode

	// ADD SECTIONS using the passed UI interface
	// CRITICAL: Initialize sections BEFORE starting lifecycle
	h.SectionBuild = h.AddSectionBUILD()
	h.AddSectionDEPLOY()

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
				return devflow.NewGitHub(logger, auth)
			}
		}
		return devflow.NewGitHub(logger)
	})

	// Initialize GoNew orchestrator with the future
	GoNew := devflow.NewGoNew(gitHandler, githubFuture, GoHandler)
	GoNew.SetLog(logger)
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

	// Call onProjectReady callback (used by daemon to set up tool proxy)
	if onProjectReady != nil {
		onProjectReady(h)
	}

	if !h.IsPartOfProject() {
		sectionWizard := h.AddSectionWIZARD(func() {
			h.OnProjectReady(&wg)
		})
		h.Tui.SetActiveTab(sectionWizard)
	} else {
		h.OnProjectReady(&wg)
	}

	if !headless {
		// Standalone mode: create and serve the project-level HTTP server on port 3030.
		// In headless/daemon-sub-project mode this is skipped to avoid a port conflict
		// with the already-running global daemon MCP.
		mcpPort := "3030"
		if p := os.Getenv("TINYWASM_MCP_PORT"); p != "" {
			mcpPort = p
		}
		mcpConfig := mcp.Config{
			Port:          mcpPort,
			ServerName:    "TinyWasm - Full-stack Go+WASM Dev Environment (Server, WASM, Assets, Browser, Deploy)",
			ServerVersion: "1.0.0",
			AppName:       "tinywasm",
		}

		// Create SSE server for log transport
		tinySSE := sse.New(&sse.Config{})
		sseHub := &sseHubAdapter{tinySSE.Server(&sse.ServerConfig{
			ChannelProvider:     &logChannelProvider{},
			ClientChannelBuffer: 256,
			HistoryReplayBuffer: 100,
			ReplayAllOnConnect:  true,
		})}

		ssePub := NewSSEPublisher(sseHub)
		h.Logger = func(messages ...any) {
			logger(messages...)
			ssePub.PublishLog(fmt.Sprint(messages...))
		}

		// Use buildProjectProviders for single source of truth
		toolHandlers := buildProjectProviders(h)
		toolHandlers = append(toolHandlers, mcpToolHandlers...)

		h.MCP = mcp.NewHandler(mcpConfig, sseHub, toolHandlers)
		h.MCP.SetLog(logger)
		h.MCP.ConfigureIDEs()
		h.MCP.SetAuth(mcp.OpenAuthorizer()) // standalone: local only, explicit opt-in

		// Register state method — same name as daemon so mcp.Client callers work identically
		h.MCP.RegisterMethod("tinywasm/state", func(_ context.Context, _ []byte) (any, error) {
			return json.RawMessage(h.Tui.GetHandlerStates()), nil
		})
		// No tinywasm/action in standalone — handlers dispatch locally via TUI

		h.Tui.AddHandler(h.MCP, colorOrangeLight, h.SectionBuild)
		SetActiveHandler(h)

		// Start HTTP server
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.MCP.Serve(h.ExitChan)
		}()

		// Start the UI
		go h.Tui.Start(&wg)
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
