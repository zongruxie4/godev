package tinywasm

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (h *handler) mcpToolGetLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Implement log retrieval from component loggers
	return mcp.NewToolResultText("Log retrieval not yet implemented"), nil
}
