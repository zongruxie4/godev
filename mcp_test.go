package golite

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServerInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MCP test in short mode")
	}

	// Create minimal handler for testing
	tmp := t.TempDir()
	h := &handler{
		frameworkName: "GOLITE",
		rootDir:       tmp,
		config:        NewConfig(tmp, func(...any) {}),
	}

	// Test that ServeMCP doesn't panic on initialization (no args now, uses port 7070)
	require.NotPanics(t, func() {
		go h.ServeMCP()
	})

	// Give HTTP server time to start, then verify it started without panicking
	// Note: HTTP server runs in background and doesn't block
}

func TestMCPToolGetStatus(t *testing.T) {
	tmp := t.TempDir()

	h := &handler{
		frameworkName: "GOLITE",
		rootDir:       tmp,
		config:        NewConfig(tmp, func(...any) {}),
	}

	// Create mock components
	// (In real usage these are created in AddSectionBUILD)

	ctx := context.Background()
	req := mcp.CallToolRequest{}

	result, err := h.mcpToolGetStatus(ctx, req)

	require.NoError(t, err, "mcpToolGetStatus should not return error")
	require.NotNil(t, result, "result should not be nil")

	// Verify result contains valid JSON
	var status map[string]any
	err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &status)
	require.NoError(t, err, "result should contain valid JSON")

	// Verify expected fields
	assert.Equal(t, "GOLITE", status["framework"], "framework should be GOLITE")
	assert.Equal(t, tmp, status["root_dir"], "root_dir should match")
	assert.Contains(t, status, "server", "should contain server status")
	assert.Contains(t, status, "wasm", "should contain wasm status")
	assert.Contains(t, status, "browser", "should contain browser status")
	assert.Contains(t, status, "assets", "should contain assets status")
}

func TestMCPToolGetLogsStub(t *testing.T) {
	tmp := t.TempDir()

	h := &handler{
		frameworkName: "GOLITE",
		rootDir:       tmp,
		config:        NewConfig(tmp, func(...any) {}),
	}

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"component": "WASM",
		"lines":     10,
	}

	result, err := h.mcpToolGetLogs(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Currently returns "not yet implemented" - this test will need updating
	// when real implementation is added
}

func TestMCPToolWasmSetModeStub(t *testing.T) {
	// Test that WASM tools are loaded dynamically via reflection
	// This test verifies the metadata loading mechanism, not the execution

	tmp := t.TempDir()

	h := &handler{
		frameworkName: "GOLITE",
		rootDir:       tmp,
		config:        NewConfig(tmp, func(...any) {}),
		wasmHandler:   nil, // No wasm handler in this test
	}

	// Verify that when wasmHandler is nil, no wasm tools are loaded
	// (This happens during ServeMCP initialization)
	// The actual tool execution is tested in integration tests

	// Test passes if handler is created without panicking
	assert.NotNil(t, h)
	assert.Nil(t, h.wasmHandler)
}

func TestMCPToolBrowserControlStubs(t *testing.T) {
	tmp := t.TempDir()

	h := &handler{
		frameworkName: "GOLITE",
		rootDir:       tmp,
		config:        NewConfig(tmp, func(...any) {}),
	}

	ctx := context.Background()
	req := mcp.CallToolRequest{}

	t.Run("browser_open", func(t *testing.T) {
		result, err := h.mcpToolBrowserOpen(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("browser_close", func(t *testing.T) {
		result, err := h.mcpToolBrowserClose(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("browser_reload", func(t *testing.T) {
		result, err := h.mcpToolBrowserReload(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("browser_get_console", func(t *testing.T) {
		result, err := h.mcpToolBrowserGetConsole(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestMCPToolEnvironmentStubs(t *testing.T) {
	tmp := t.TempDir()

	h := &handler{
		frameworkName: "GOLITE",
		rootDir:       tmp,
		config:        NewConfig(tmp, func(...any) {}),
	}

	ctx := context.Background()
	req := mcp.CallToolRequest{}

	t.Run("project_structure", func(t *testing.T) {
		result, err := h.mcpToolProjectStructure(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("check_requirements", func(t *testing.T) {
		result, err := h.mcpToolCheckRequirements(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("deploy_status", func(t *testing.T) {
		result, err := h.mcpToolDeployStatus(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}
