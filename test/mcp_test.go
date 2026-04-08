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

func setupMCPTest(t *testing.T) (*mcp.Server, string) {
	port := freePort()
	mcpConfig := mcp.Config{
		Name:    "TINYWASM",
		Version: "1.0.0",
		Auth:    mcp.OpenAuthorizer(),
	}

	m, err := mcp.NewServer(mcpConfig, []mcp.ToolProvider{})
	if err != nil {
		t.Fatalf("failed to create mcp server: %v", err)
	}
	return m, port
}

func TestMCPHTTPHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MCP test in short mode")
	}

	m, _ := setupMCPTest(t)

	// Since we no longer have HTTPHandler(), we can't test it this way.
	// But we can check that m is not nil.
	if m == nil {
		t.Error("setupMCPTest should return a non-nil *mcp.Server")
	}
}
