package test

import (
	"net/http"
	"testing"

	"github.com/tinywasm/mcp"
)

// channelProvider implements sse.ChannelProvider
type channelProvider struct{}

func (p *channelProvider) ResolveChannels(r *http.Request) ([]string, error) {
	return []string{"logs"}, nil
}

func setupMCPTest(t *testing.T) (*mcp.Handler, string) {
	port := freePort()
	mcpConfig := mcp.Config{
		Port:          port,
		ServerName:    "TINYWASM",
		ServerVersion: "1.0.0",
		AppName:       "tinywasm",
	}

	m := mcp.NewHandler(mcpConfig, []mcp.ToolProvider{})
	return m, port
}

func TestMCPHTTPHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MCP test in short mode")
	}

	m, _ := setupMCPTest(t)

	// Test that HTTPHandler returns a valid http.Handler
	handler := m.HTTPHandler()
	if handler == nil {
		t.Error("HTTPHandler should return a non-nil http.Handler")
	}
}

func TestMCPConfigureIDEs(t *testing.T) {
	m, _ := setupMCPTest(t)

	// Should not panic even if IDEs aren't installed
	m.ConfigureIDEs()
}
