package golite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServeMCP starts the Model Context Protocol server for LLM integration
// This allows AI assistants to monitor and control the GoLite development environment
// This function blocks on stdio, so it should be called in a goroutine
func (h *handler) ServeMCP() {
	// Create MCP server with tool capabilities
	s := server.NewMCPServer(
		"GoLite Development Assistant",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// === STATUS & MONITORING TOOLS ===

	s.AddTool(mcp.NewTool("golite_status",
		mcp.WithDescription("Get comprehensive status of GoLite environment (server, WASM, browser, assets)"),
	), h.mcpToolGetStatus)

	s.AddTool(mcp.NewTool("golite_get_logs",
		mcp.WithDescription("Retrieve recent logs from specific component"),
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

	s.AddTool(mcp.NewTool("wasm_set_mode",
		mcp.WithDescription("Change WASM compilation mode (LARGE=Go std, MEDIUM=TinyGo optimized, SMALL=TinyGo compact)"),
		mcp.WithString("mode",
			mcp.Required(),
			mcp.Description("Compilation mode to set"),
			mcp.Enum("LARGE", "L", "MEDIUM", "M", "SMALL", "S"),
		),
	), h.mcpToolWasmSetMode)

	s.AddTool(mcp.NewTool("wasm_recompile",
		mcp.WithDescription("Force WASM recompilation with current mode"),
	), h.mcpToolWasmRecompile)

	s.AddTool(mcp.NewTool("wasm_get_size",
		mcp.WithDescription("Get current WASM file size and comparison across modes"),
	), h.mcpToolWasmGetSize)

	// === BROWSER CONTROL TOOLS ===

	s.AddTool(mcp.NewTool("browser_open",
		mcp.WithDescription("Open development browser pointing to local server"),
	), h.mcpToolBrowserOpen)

	s.AddTool(mcp.NewTool("browser_close",
		mcp.WithDescription("Close development browser and cleanup resources"),
	), h.mcpToolBrowserClose)

	s.AddTool(mcp.NewTool("browser_reload",
		mcp.WithDescription("Reload browser page to see latest changes"),
	), h.mcpToolBrowserReload)

	s.AddTool(mcp.NewTool("browser_get_console",
		mcp.WithDescription("Get browser console logs (errors, warnings, logs)"),
		mcp.WithString("level",
			mcp.DefaultString("all"),
			mcp.Description("Log level filter"),
			mcp.Enum("all", "error", "warning", "log"),
		),
		mcp.WithNumber("lines",
			mcp.DefaultNumber(50),
			mcp.Description("Number of recent entries to retrieve"),
		),
	), h.mcpToolBrowserGetConsole)

	// === DEPLOYMENT TOOLS ===

	s.AddTool(mcp.NewTool("deploy_status",
		mcp.WithDescription("Get Cloudflare deployment configuration and status"),
	), h.mcpToolDeployStatus)

	// === ENVIRONMENT TOOLS ===

	s.AddTool(mcp.NewTool("project_structure",
		mcp.WithDescription("Get project directory structure with file counts"),
	), h.mcpToolProjectStructure)

	s.AddTool(mcp.NewTool("check_requirements",
		mcp.WithDescription("Verify development environment (Go, TinyGo, Chrome)"),
	), h.mcpToolCheckRequirements)

	// Start the stdio server (blocks on stdio)
	// ServeMCP is already called in a goroutine from start.go
	if err := server.ServeStdio(s); err != nil {
		// Log error but don't crash golite
		if h.config != nil && h.config.logger != nil {
			h.config.logger("MCP server error:", err)
		}
	}
}

// === TOOL IMPLEMENTATIONS ===

func (h *handler) mcpToolGetStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := map[string]interface{}{
		"framework": h.frameworkName,
		"root_dir":  h.rootDir,
		"server": map[string]interface{}{
			"running":    h.serverHandler != nil,
			"port":       h.config.ServerPort(),
			"output_dir": h.config.DeployAppServerDir(),
		},
		"wasm": map[string]interface{}{
			"output_dir": h.config.WebPublicDir(),
			// TODO: Get current mode from wasmHandler
		},
		"browser": map[string]interface{}{
			// TODO: Get isOpen status from browser
			"url": fmt.Sprintf("http://localhost:%s", h.config.ServerPort()),
		},
		"assets": map[string]interface{}{
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
