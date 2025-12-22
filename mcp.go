package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPPort is the fixed port for TinyWasm's MCP HTTP server
// Using port 3030 (above 1024 to avoid needing root privileges)
const MCPPort = "3030"

// ServeMCP starts the Model Context Protocol server for LLM integration via HTTP
// Runs on port 3030 without conflicts with the UI
// Listens to exitChan to shutdown gracefully
func (h *handler) ServeMCP() {
	// Create MCP server with tool capabilities
	s := server.NewMCPServer(
		"TinyWasm - Full-stack Go+WASM Dev Environment (Server, WASM, Assets, Browser, Deploy)",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// === STATUS & MONITORING TOOLS ===

	s.AddTool(mcp.NewTool("golite_status",
		mcp.WithDescription("Get comprehensive status of TinyWasm full-stack dev environment: Go server (running/port), WASM compilation (output dir), browser (URL), and asset watching. Use this first to understand the current state of the development environment."),
	), h.mcpToolGetStatus)

	s.AddTool(mcp.NewTool("golite_get_logs",
		mcp.WithDescription("Retrieve recent logs from specific component (WASM compiler, Go server, asset minifier, file watcher, browser, or Cloudflare deploy) to debug build issues or track changes."),
		mcp.WithString("component",
			mcp.Required(),
			mcp.Description("Component to get logs from"),
			mcp.Enum("WASM", "SERVER", "ASSETS", "WATCH", "BROWSER", "CLOUDFLARE"),
		),
		mcp.WithNumber("lines",
			mcp.DefaultNumber(50),
			mcp.Description("Number of recent log lines to retrieve"),
		),
	), h.mcpToolGetLogs)

	// === BUILD CONTROL TOOLS ===

	// Load WASM tools from TinyWasm metadata using reflection
	if h.wasmClient != nil {
		if tools, err := mcpToolsFromHandler(h.wasmClient); err == nil {
			for _, toolMeta := range tools {
				tool := buildMCPTool(toolMeta)
				// Use generic executor - no need to know tool names or implementations
				s.AddTool(*tool, h.mcpExecuteTool(toolMeta.Execute))
			}
		} else {
			// Safe logging - check if config is initialized
			if h.config != nil && h.config.logger != nil {
				h.config.logger("Warning: Failed to load WASM tools:", err)
			}
		}
	}

	// === BROWSER CONTROL TOOLS ===

	// Load Browser tools from DevBrowser metadata using reflection
	if h.browser != nil {
		if tools, err := mcpToolsFromHandler(h.browser); err == nil {
			for _, toolMeta := range tools {
				tool := buildMCPTool(toolMeta)
				// Use generic executor - no need to know tool names or implementations
				s.AddTool(*tool, h.mcpExecuteTool(toolMeta.Execute))
			}
		} else {
			// Safe logging - check if config is initialized
			if h.config != nil && h.config.logger != nil {
				h.config.logger("Warning: Failed to load Browser tools:", err)
			}
		}
	}

	// === DEPLOYMENT TOOLS ===

	s.AddTool(mcp.NewTool("deploy_status",
		mcp.WithDescription("Get Cloudflare Workers deployment configuration and status (for deploying the full-stack Go+WASM app to production edge network)."),
	), h.mcpToolDeployStatus)

	// === ENVIRONMENT TOOLS ===

	s.AddTool(mcp.NewTool("project_structure",
		mcp.WithDescription("Get Go project directory structure with file counts (shows cmd/appserver for backend, cmd/webclient for WASM frontend, deploy dirs, etc)."),
	), h.mcpToolProjectStructure)

	s.AddTool(mcp.NewTool("check_requirements",
		mcp.WithDescription("Verify development environment has required tools installed: Go compiler (backend), TinyGo compiler (WASM frontend), and Chrome browser (testing)."),
	), h.mcpToolCheckRequirements)

	// Start MCP HTTP server (runs concurrently with UI)
	// Use stateless mode: no session management needed for single developer
	httpServer := server.NewStreamableHTTPServer(s,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true), // No session IDs required
	)

	// Store reference for shutdown
	h.mcpServer = httpServer

	// Safe logging - check if config is initialized
	if h.config != nil && h.config.logger != nil {
		h.config.logger("Starting MCP HTTP server on port", MCPPort)
		h.config.logger("MCP endpoint: http://localhost:" + MCPPort + "/mcp")
	}

	// Start server in goroutine (it blocks)
	go func() {
		if err := httpServer.Start(":" + MCPPort); err != nil {
			if h.config != nil && h.config.logger != nil {
				h.config.logger("MCP HTTP server stopped:", err)
			}
		}
	}()

	// Wait for exit signal from UI
	// When channel is closed, ok will be false
	_, ok := <-h.exitChan
	if !ok {
		// Channel closed, shutdown gracefully
		if h.config != nil && h.config.logger != nil {
			h.config.logger("Shutting down MCP server...")
		}
		ctx := context.Background()
		if err := httpServer.Shutdown(ctx); err != nil {
			if h.config != nil && h.config.logger != nil {
				h.config.logger("Error shutting down MCP server:", err)
			}
		}
	}
}

// === TOOL IMPLEMENTATIONS ===

func (h *handler) mcpToolGetStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if handler is fully initialized
	if h.config == nil {
		return mcp.NewToolResultError("TinyWasm is still initializing. Please try again in a moment."), nil
	}

	status := map[string]any{
		"framework": h.frameworkName,
		"root_dir":  h.rootDir,
		"server": map[string]any{
			"running":    h.serverHandler != nil,
			"port":       h.config.ServerPort(),
			"output_dir": h.config.DeployAppServerDir(),
		},
		"wasm": map[string]any{
			"output_dir": h.config.WebPublicDir(),
			// TODO: Get current mode from wasmClient
		},
		"browser": map[string]any{
			// TODO: Get isOpen status from browser
			"url": fmt.Sprintf("http://localhost:%s", h.config.ServerPort()),
		},
		"assets": map[string]any{
			"watching":   true,
			"public_dir": h.config.WebPublicDir(),
		},
	}

	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal status: " + err.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (h *handler) mcpToolProjectStructure(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Scan directory structure
	return mcp.NewToolResultText("Project structure scan not yet implemented"), nil
}

func (h *handler) mcpToolCheckRequirements(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Check for go, tinygo, chrome executables
	return mcp.NewToolResultText("Requirements check not yet implemented"), nil
}
