package golite

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

func (h *handler) mcpToolWasmSetMode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Implement mode change via wasmHandler.Change()
	return mcp.NewToolResultText("WASM mode change not yet implemented"), nil
}

func (h *handler) mcpToolWasmRecompile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Implement forced recompilation
	return mcp.NewToolResultText("WASM recompile not yet implemented"), nil
}

func (h *handler) mcpToolWasmGetSize(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: Implement size retrieval
	return mcp.NewToolResultText("WASM size info not yet implemented"), nil
}
