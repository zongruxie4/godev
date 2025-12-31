# TinyWasm MCP Server

LLM control for the TinyWasm environment.

## Quick Start
1. Run `tinywasm`.
2. Connect your IDE/LLM to `http://localhost:3030/mcp`.

### Claude Desktop Config
```json
{
  "mcpServers": {
    "tinywasm": { "url": "http://localhost:3030/mcp" }
  }
}
```

## Tools
- `golite_status`: Core env status.
- `wasm_set_mode`: Change WASM mode (S/M/L).
- `wasm_recompile`: Force rebuild.
- `browser_reload`: Reload DevBrowser.
- `golite_get_logs`: Component logs.

*Full list available via `tools/list` endpoint.*

## Advanced Details
See [mcpserve README](../../mcpserve/README.md) for development and architecture.
