package golite

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (h *handler) mcpToolBrowserOpen(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Call h.browser.OpenBrowser()
	return mcp.NewToolResultText("Browser open not yet implemented"), nil
}

func (h *handler) mcpToolBrowserClose(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Call h.browser.CloseBrowser()
	return mcp.NewToolResultText("Browser close not yet implemented"), nil
}

func (h *handler) mcpToolBrowserReload(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Call h.browser.Reload()
	return mcp.NewToolResultText("Browser reload not yet implemented"), nil
}

func (h *handler) mcpToolBrowserGetConsole(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Implement console log retrieval via CDP
	return mcp.NewToolResultText("Browser console logs not yet implemented"), nil
}
