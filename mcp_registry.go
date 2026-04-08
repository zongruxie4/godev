package app

import (
	"sync"

	"github.com/tinywasm/mcp"
)

// ProjectToolProxy implements mcp.ToolProvider and is registered once with the daemon.
// It delegates to the current active project's tool providers atomically.
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

// Tools implements mcp.ToolProvider.
// Always reflects the current active project's tools.
func (p *ProjectToolProxy) Tools() []mcp.Tool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var tools []mcp.Tool
	for _, provider := range p.active {
		if provider != nil {
			tools = append(tools, provider.Tools()...)
		}
	}
	return tools
}

// BrowserAdapter wraps BrowserInterface to satisfy mcp.ToolProvider
type BrowserAdapter struct {
	BrowserInterface
}

func (a *BrowserAdapter) Tools() []mcp.Tool {
	// Re-map tools from old to new if necessary.
	// We know devbrowser.GetMCPTools() returns []mcp.Tool but probably the OLD mcp.Tool struct.
	// HOWEVER, since devbrowser v0.3.19 ALREADY depends on mcp v0.1.1, its GetMCPTools
	// should already return the NEW mcp.Tool struct if it compiles.
	// Let's assume it returns the new struct but with old method name.
	return a.BrowserInterface.GetMCPTools()
}

// buildProjectProviders returns the ordered list of tool providers for a project.
// This is the single place where project-level tool registration order is defined.
func buildProjectProviders(h *Handler) []mcp.ToolProvider {
	providers := []mcp.ToolProvider{h} // app_rebuild tool first
	// We might need to cast or ensure WasmClient and Browser implement ToolProvider
	if h.WasmClient != nil {
		if tp, ok := any(h.WasmClient).(mcp.ToolProvider); ok {
			providers = append(providers, tp)
		}
	}
	if h.Browser != nil {
		providers = append(providers, &BrowserAdapter{h.Browser})
	}
	return providers
}
