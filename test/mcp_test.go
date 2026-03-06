package test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tinywasm/mcp"
	"github.com/tinywasm/sse"
)

// stubTUI implements mcp.tuiRefresher
type stubTUI struct{}

func (s *stubTUI) RefreshUI() {}

// channelProvider implements sse.ChannelProvider
type channelProvider struct{}

func (p *channelProvider) ResolveChannels(r *http.Request) ([]string, error) {
	return []string{"logs"}, nil
}

func setupMCPTest(t *testing.T) (*mcp.Handler, chan bool, string) {
	ExitChan := make(chan bool)
	port := freePort()
	mcpConfig := mcp.Config{
		Port:          port,
		ServerName:    "TINYWASM",
		ServerVersion: "1.0.0",
	}

	// Create SSE server for tests
	tinySSE := sse.New(&sse.Config{})
	sseHub := tinySSE.Server(&sse.ServerConfig{
		ChannelProvider:     &channelProvider{},
		ClientChannelBuffer: 256,
		HistoryReplayBuffer: 100,
		ReplayAllOnConnect:  true,
	})

	m := mcp.NewHandler(mcpConfig, []mcp.ToolProvider{}, &stubTUI{}, sseHub, ExitChan)
	return m, ExitChan, port
}

func TestMCPServerInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MCP test in short mode")
	}

	m, ExitChan, port := setupMCPTest(t)
	startupErrors := make(chan error, 1)

	// Test that Serve doesn't panic on initialization
	go func() {
		// Catch any panic during Serve
		defer func() {
			if r := recover(); r != nil {
				startupErrors <- fmt.Errorf("Serve panicked: %v", r)
				t.Errorf("Serve panicked: %v", r)
			}
		}()
		m.Serve()
	}()

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
		if resp.StatusCode != 200 {
			t.Errorf("MCP server should respond with 200, got %v", resp.StatusCode)
		}
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
