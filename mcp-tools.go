package app

import (
	"github.com/tinywasm/mcpserve"
)

// GetMCPToolsMetadata returns metadata for all Handler MCP tools
func (h *Handler) GetMCPToolsMetadata() []mcpserve.ToolMetadata {
	return []mcpserve.ToolMetadata{
		{
			Name: "app_rebuild",
			Description: "Trigger a full site recompilation (WASM, assets) and reload the environment. " +
				"Use this after making code changes to ensure they are applied.",
			Parameters: []mcpserve.ParameterMetadata{},
			Execute: func(args map[string]any) {
				h.Logger("Manual rebuild triggered via MCP...")
				if h.WasmClient != nil {
					if err := h.WasmClient.RecompileMainWasm(); err != nil {
						h.Logger("Rebuild failed:", err)
						return
					}

					// Trigger update hooks (Asset refresh, Server restart, Browser reload)
					// These are wired in section-build.go
					if h.WasmClient.OnWasmExecChange != nil {
						h.WasmClient.OnWasmExecChange()
					}
					h.Logger("Rebuild completed successfully")
				} else {
					h.Logger("WasmClient is not initialized")
				}
			},
		},
	}
}
