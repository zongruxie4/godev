package app

import (
	"sync"

	"github.com/tinywasm/mcp"
)

// ProjectToolProxy implements mcp.ToolProvider and is registered once with the daemon.
// It delegates to the current active project's tool providers atomically.
// This solves the lifecycle mismatch: the MCP handler is created once at daemon startup,
// but project-level tool providers (Handler, Browser, WasmClient) are created per project.
type ProjectToolProxy struct {
	mu     sync.RWMutex
	active []mcp.ToolProvider
}

// NewProjectToolProxy creates a new ProjectToolProxy
func NewProjectToolProxy() *ProjectToolProxy {
	return &ProjectToolProxy{
		active: []mcp.ToolProvider{},
	}
}

// SetActive updates the active project's tool providers.
// Call with no args to clear (project stopped).
func (p *ProjectToolProxy) SetActive(providers ...mcp.ToolProvider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(providers) == 0 {
		p.active = []mcp.ToolProvider{}
	} else {
		p.active = providers
	}
}

// GetMCPTools implements mcp.ToolProvider.
// Always reflects the current active project's tools.
func (p *ProjectToolProxy) GetMCPTools() []mcp.Tool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var tools []mcp.Tool
	for _, provider := range p.active {
		tools = append(tools, provider.GetMCPTools()...)
	}
	return tools
}

// buildProjectProviders returns the ordered list of tool providers for a project.
// This is the single place where project-level tool registration order is defined.
func buildProjectProviders(h *Handler) []mcp.ToolProvider {
	providers := []mcp.ToolProvider{h} // app_rebuild tool first
	if h.WasmClient != nil {
		providers = append(providers, h.WasmClient)
	}
	if h.Browser != nil {
		providers = append(providers, h.Browser)
	}
	return providers
}
