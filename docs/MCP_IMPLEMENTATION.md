# TinyWasm MCP Implementation Guide

## Quick Start

The MCP server is automatically started when you run `golite`. No additional configuration needed.

## Testing the MCP Server

### Using MCP Inspector

1. Install the MCP inspector:
```bash
npm install -g @modelcontextprotocol/inspector
```

2. Run golite in your project:
```bash
cd your-go-project
golite
```

3. In another terminal, connect the inspector:
```bash
mcp-inspector golite
```

### Using Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "golite": {
      "command": "golite",
      "args": [],
      "env": {}
    }
  }
}
```

Then restart Claude Desktop. TinyWasm tools will appear in the tools menu.

## Implementation Status

### ✅ Completed
- [x] MCP server structure
- [x] All 13 tool definitions
- [x] Auto-start with golite
- [x] Basic status tool (partial)

### 🚧 In Progress
- [ ] Full status tool implementation
- [ ] Log buffer system
- [ ] WASM mode control integration
- [ ] Browser control integration
- [ ] Console log capture via CDP

### 📋 TODO
- [ ] Environment requirements check
- [ ] Project structure scanner
- [ ] Deployment status
- [ ] WASM size analyzer
- [ ] Comprehensive tests

## Architecture

```
┌─────────────────────────────────────────────┐
│ LLM Client (Claude, ChatGPT, etc.)         │
└─────────────────┬───────────────────────────┘
                  │ MCP Protocol (stdio)
┌─────────────────▼───────────────────────────┐
│ TinyWasm MCP Server (mcp.go)                  │
│ - 13 tool handlers                          │
│ - JSON-RPC 2.0 interface                    │
└─────────────────┬───────────────────────────┘
                  │
        ┌─────────┴─────────┬─────────────┬───────────┐
        ▼                   ▼             ▼           ▼
┌───────────────┐  ┌──────────────┐  ┌────────┐  ┌────────┐
│ handler       │  │ wasmHandler  │  │browser │  │watcher │
│ (golite core) │  │ (tinywasm)   │  │(devbr) │  │(watch) │
└───────────────┘  └──────────────┘  └────────┘  └────────┘
```

## Implementation Pattern

Each tool follows this pattern:

```go
func (h *handler) mcpToolName(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 1. Extract arguments
    arg, err := req.RequireString("arg_name")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    // 2. Perform operation on handler components
    result := h.someComponent.DoSomething(arg)
    
    // 3. Format response as JSON
    data := map[string]any{
        "status": "success",
        "result": result,
    }
    
    jsonData, _ := json.Marshal(data)
    return mcp.NewToolResultText(string(jsonData)), nil
}
```

## Adding Log Buffers

To support `golite_get_logs`, each component logger needs a buffer:

```go
// In logs.go or new file
type LogBuffer struct {
    entries []LogEntry
    maxSize int
    mu      sync.RWMutex
}

type LogEntry struct {
    Timestamp time.Time
    Message   string
    Level     string
}

func NewLogBuffer(maxSize int) *LogBuffer {
    return &LogBuffer{
        entries: make([]LogEntry, 0, maxSize),
        maxSize: maxSize,
    }
}

func (lb *LogBuffer) Add(message string) {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    
    entry := LogEntry{
        Timestamp: time.Now(),
        Message:   message,
        Level:     "info",
    }
    
    lb.entries = append(lb.entries, entry)
    if len(lb.entries) > lb.maxSize {
        lb.entries = lb.entries[1:]
    }
}

func (lb *LogBuffer) GetRecent(n int) []LogEntry {
    lb.mu.RLock()
    defer lb.mu.RUnlock()
    
    if n > len(lb.entries) {
        n = len(lb.entries)
    }
    
    start := len(lb.entries) - n
    return lb.entries[start:]
}
```

Then wrap existing loggers:

```go
type LoggerWithBuffer struct {
    originalLogger func(messages ...any)
    buffer         *LogBuffer
}

func (l *LoggerWithBuffer) Log(messages ...any) {
    msg := fmt.Sprint(messages...)
    l.buffer.Add(msg)
    l.originalLogger(messages...)
}
```

## Browser Console Integration

To implement `browser_get_console`, extend devbrowser with CDP:

```go
// In devbrowser package
type ConsoleLog struct {
    Level     string    `json:"level"`
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
}

func (b *DevBrowser) GetConsoleLogs(level string, lines int) []ConsoleLog {
    if b.page == nil {
        return nil
    }
    
    // Use rod's Eval to get console history
    // This requires setting up console message listeners
    // during CreateBrowserContext()
    
    // Pseudo-code:
    // result, _ := b.page.Eval(`() => {
    //     return window._consoleHistory || []
    // }`)
    
    return []ConsoleLog{} // placeholder
}

func (b *DevBrowser) setupConsoleCapture() {
    // In CreateBrowserContext, inject JavaScript:
    // window._consoleHistory = []
    // const originalLog = console.log
    // console.log = function(...args) {
    //     window._consoleHistory.push({level: 'log', message: args.join(' '), time: Date.now()})
    //     originalLog.apply(console, args)
    // }
    // Similar for console.error, console.warn
}
```

## Testing Tools

### Manual Test

```bash
# Terminal 1: Start golite
cd test-project
golite

# Terminal 2: Send MCP request
echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"golite_status","arguments":{}},"id":1}' | golite

# You should receive JSON-RPC response with current status
```

### Automated Test

```go
func TestMCPToolStatus(t *testing.T) {
    h := &handler{
        frameworkName: "GOLITE",
        rootDir: t.TempDir(),
        config: NewConfig(t.TempDir(), func(...any){}),
    }
    
    req := mcp.CallToolRequest{}
    result, err := h.mcpToolGetStatus(context.Background(), req)
    
    require.NoError(t, err)
    require.NotNil(t, result)
    
    // Parse result and verify fields
}
```

## Deployment

The MCP server is embedded in golite - no separate deployment needed. Users just run:

```bash
go install github.com/tinywasm/tinywasm/cmd/golite@latest
```

And MCP is automatically available.

## Debugging

Enable MCP debug output:

```bash
export MCP_DEBUG=1
golite
```

This will show all MCP protocol messages in the logs.

## Next Steps

1. Implement log buffer system
2. Connect WASM mode control to wasmHandler.Change()
3. Add browser console capture
4. Implement project structure scanner
5. Add comprehensive tests
6. Document in main README.md

## Resources

- [MCP Specification](https://modelcontextprotocol.io/)
- [mcp-go Documentation](https://github.com/mark3labs/mcp-go)
- [TinyWasm Documentation](../README.md)
