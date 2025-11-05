package golite

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (h *handler) mcpToolDeployStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Get status from h.deployCloudflare
	return mcp.NewToolResultText("Deploy status not yet implemented"), nil
}
