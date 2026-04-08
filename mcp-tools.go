package app

import (
	"errors"

	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
)

// Tools returns metadata for all Handler MCP tools
func (h *Handler) Tools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name: "app_rebuild",
			Description: "Trigger a full site recompilation (WASM, assets) and reload the environment. " +
				"Use this after making code changes to ensure they are applied.",
			InputSchema: `{"type":"object","properties":{}}`,
			Resource:    "app",
			Action:      'u',
			Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
				h.Logger("Manual rebuild triggered via MCP...")
				if h.WasmClient != nil {
					if err := h.WasmClient.RecompileMainWasm(); err != nil {
						h.Logger("Rebuild failed:", err)
						return nil, err
					}

					// Trigger update hooks (Asset refresh, Server restart, Browser reload)
					if h.WasmClient.OnWasmExecChange != nil {
						h.WasmClient.OnWasmExecChange()
					}
					h.Logger("Rebuild completed successfully")
					return mcp.Text("Rebuild completed successfully"), nil
				} else {
					h.Logger("WasmClient is not initialized")
					return nil, errors.New("WasmClient not initialized")
				}
			},
		},
	}
}
