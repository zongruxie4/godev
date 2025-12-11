package tinywasm

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

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
	exitChan := make(chan bool)

	// Capture any errors during startup
	startupErrors := make(chan error, 1)

	h := &handler{
		frameworkName: "GOLITE",
		rootDir:       tmp,
		config: NewConfig(tmp, func(messages ...any) {
			// Log to test output
			t.Log(messages...)
		}),
		exitChan: exitChan,
	}

	// Test that ServeMCP doesn't panic on initialization
	require.NotPanics(t, func() {
		go func() {
			// Catch any panic during ServeMCP
			defer func() {
				if r := recover(); r != nil {
					startupErrors <- assert.AnError
					t.Errorf("ServeMCP panicked: %v", r)
				}
			}()
			h.ServeMCP()
		}()
	})

	// Give HTTP server time to start
	time.Sleep(200 * time.Millisecond)

	// Check if there was a startup error
	select {
	case err := <-startupErrors:
		t.Fatalf("ServeMCP failed to start: %v", err)
	default:
		// No error, continue
	}

	// Try to connect to verify server is actually running
	resp, err := http.Post("http://localhost:"+MCPPort+"/mcp",
		"application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))

	if err != nil {
		t.Errorf("Failed to connect to MCP server: %v", err)
		t.Error("MCP server should be running and accepting connections")
	} else {
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode, "MCP server should respond with 200")
	}

	// Cleanup: close exit channel to stop server
	close(exitChan)
	time.Sleep(50 * time.Millisecond)
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
