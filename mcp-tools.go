package app

import (
	"github.com/tinywasm/mcp"
)

// Tools returns metadata for all Handler MCP tools.
// app_rebuild is intentionally not exposed: tinywasm recompiles automatically on file change.
func (h *Handler) Tools() []mcp.Tool {
	return []mcp.Tool{}
}
