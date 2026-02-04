package test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinywasm/mcpserve"
)

func setupMCPTest(t *testing.T) (*mcpserve.Handler, chan bool, string) {
	ExitChan := make(chan bool)
	port := freePort()
	mcpConfig := mcpserve.Config{
		Port:          port,
		ServerName:    "TINYWASM",
		ServerVersion: "1.0.0",
	}
	m := mcpserve.NewHandler(mcpConfig, nil, nil, ExitChan)
	return m, ExitChan, port
}

func TestMCPServerInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MCP test in short mode")
	}

	m, ExitChan, port := setupMCPTest(t)
	startupErrors := make(chan error, 1)

	// Test that Serve doesn't panic on initialization
	require.NotPanics(t, func() {
		go func() {
			// Catch any panic during Serve
			defer func() {
				if r := recover(); r != nil {
					startupErrors <- assert.AnError
					t.Errorf("Serve panicked: %v", r)
				}
			}()
			m.Serve()
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
	resp, err := http.Post("http://localhost:"+port+"/mcp",
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
	close(ExitChan)
	time.Sleep(50 * time.Millisecond)
}

func TestMCPConfigureIDEs(t *testing.T) {
	m, ExitChan, _ := setupMCPTest(t)
	defer close(ExitChan)

	// Should not panic even if IDEs aren't installed
	m.ConfigureIDEs()
}
