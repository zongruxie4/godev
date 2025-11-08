# GoLite MCP Server

**LLM control for GoLite development environment**

## Quick Start

```bash
golite  # Starts UI + MCP server on http://localhost:3030/mcp
```

### Connect from Claude Desktop

```json
{
  "mcpServers": {
    "golite": {
      "url": "http://localhost:3030/mcp"
    }
  }
}
```

---

## Available Tools

### Status & Monitoring
- **`golite_status`** - Get complete environment status (server, WASM, browser, assets)
- **`golite_get_logs`** - Get recent logs from component (WASM, SERVER, ASSETS, WATCH, BROWSER, CLOUDFLARE)

### Build Control
- **`wasm_set_mode`** - Change WASM mode (LARGE/L, MEDIUM/M, SMALL/S)
- **`wasm_recompile`** - Force WASM rebuild
- **`wasm_get_size`** - Get current WASM size + comparisons

### Browser Control
- **`browser_open`** - Open development browser
- **`browser_close`** - Close browser
- **`browser_reload`** - Reload page
- **`browser_get_console`** - Get console logs (filter: all/error/warning/log)

### Deployment
- **`deploy_status`** - Get Cloudflare deployment config

### Environment
- **`project_structure`** - Get project directory structure
- **`check_requirements`** - Verify Go, TinyGo, Chrome installation

---

## Examples

### Check Status

```bash
curl -X POST http://localhost:3030/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "golite_status",
      "arguments": {}
    }
  }' | jq -r '.result.content[0].text' | jq '.'
```

### Change WASM Mode

```bash
curl -X POST http://localhost:3030/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "wasm_set_mode",
      "arguments": {
        "mode": "SMALL"
      }
    }
  }' | jq -r '.result.content[0].text'
```

### Get Browser Console Errors

```bash
curl -X POST http://localhost:3030/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "browser_get_console",
      "arguments": {
        "level": "error"
      }
    }
  }' | jq -r '.result.content[0].text'
```

---

**Note**: Server runs in stateless mode - no session management needed. Just send requests directly to `http://localhost:3030/mcp`

