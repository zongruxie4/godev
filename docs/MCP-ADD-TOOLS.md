# MCP Tools Integration Guide

## Overview
This guide explains how to add new MCP (Model Context Protocol) tools to TinyWasm using a reflection-based metadata pattern with **complete decoupling** between the MCP framework layer and domain handler logic.

## Architecture Principles

**Key Rule:** Domain handlers (TinyWasm, DevBrowser, etc.) should NEVER import or know about MCP concepts.

**Pattern:** Handlers expose all their tools in a single method that returns `[]ToolMetadata` with embedded execution functions. TinyWasm uses a **generic executor** that works for ALL tools without knowing their names or implementations.

## Adding MCP Tools to a Handler

### Step 1: Define Metadata Structures in Handler Package

In your handler package (e.g., `tinywasm/`), define metadata structures with execution function:

```go
// ToolExecutor defines how a tool should be executed
type ToolExecutor func(args map[string]any, progress chan<- string)

// ToolMetadata describes a tool's interface and execution
type ToolMetadata struct {
    Name        string
    Description string
    Parameters  []ParameterMetadata
    Execute     ToolExecutor // Handler provides execution function
}

// ParameterMetadata describes a tool parameter
type ParameterMetadata struct {
    Name        string
    Description string
    Required    bool
    Type        string // "string", "number", "boolean"
    EnumValues  []string
    Default     any
}
```

### Step 2: Implement GetMCPToolsMetadata Method with Execution Logic

Add a single method that returns ALL tools with their execution functions:

```go
// GetMCPToolsMetadata returns metadata for all TinyWasm MCP tools
func (w *TinyWasm) GetMCPToolsMetadata() []ToolMetadata {
    return []ToolMetadata{
        {
            Name: "wasm_set_mode",
            Description: "Change WASM compilation mode. L=LARGE (~2MB), M=MEDIUM (~500KB), S=SMALL (~200KB)",
            Parameters: []ParameterMetadata{
                {
                    Name:        "mode",
                    Description: "Compilation mode: L, M, or S",
                    Required:    true,
                    Type:        "string",
                    EnumValues:  []string{"L", "M", "S"},
                },
            },
            Execute: func(args map[string]any, progress chan<- string) {
                // Extract and validate parameters
                modeValue, ok := args["mode"]
                if !ok {
                    progress <- "error: missing required parameter 'mode'"
                    return
                }
                
                mode, ok := modeValue.(string)
                if !ok {
                    progress <- "error: parameter 'mode' must be a string"
                    return
                }
                
                // Domain-specific logic stays here
                // Domain functions should accept a send-only string channel for progress updates.
                w.Change(mode, progress)
                progress <- "mode changed"
            },
        },
        {
            Name:        "wasm_recompile",
            Description: "Force immediate WASM recompilation with current mode",
            Parameters:  []ParameterMetadata{}, // No parameters
            Execute: func(args map[string]any, progress chan<- string) {
                if err := w.RecompileMainWasm(); err != nil {
                    progress <- fmt.Sprintf("recompilation failed: %v", err)
                    return
                }
                progress <- "WASM recompiled successfully"
            },
        },
    }
}
```

**Key Points:**
- Method MUST be named exactly `GetMCPToolsMetadata()` and return `[]ToolMetadata`
- Each tool includes its own `Execute` function with all domain logic
- `Execute` receives `args` (parameters) and `progress` (send-only channel for messages)
- `Execute` does NOT return `error`; use the `progress` channel to report success or errors

### Step 3: TinyWasm Registration (Already Done - No Code Needed!)

In `tinywasm/mcp.go`, tools are loaded automatically with a **generic executor**:

```go
// Load WASM tools from TinyWasm metadata using reflection
if h.wasmHandler != nil {
    if tools, err := mcpToolsFromHandler(h.wasmHandler); err == nil {
        for _, toolMeta := range tools {
            tool := buildMCPTool(toolMeta)
            // Generic executor works for ALL tools - no switch needed!
            s.AddTool(*tool, h.mcpExecuteTool(toolMeta.Execute))
        }
    }
}
```

**Notice:** 
- ✅ No `switch` statement on tool names
- ✅ No individual handler functions per tool
- ✅ TinyWasm doesn't know anything about "wasm_set_mode" or any specific tool
- ✅ Just loop and register - works for ANY handler


## How the Generic Executor Works

The `mcpExecuteTool` function in `tinywasm/mcp-executor.go` is a **universal handler** that works for ANY tool:

```go
func (h *handler) mcpExecuteTool(executor ToolExecutor) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // 1. Extract arguments (generic)
        args, _ := req.Params.Arguments.(map[string]any)
        
        // 2. Collect progress messages (generic) via a channel
        txtOut := []string{}
        ch := make(chan string, 16)
        done := make(chan struct{})

        // Run executor in a goroutine, close channel when done
        go func() {
            executor(args, ch)
            close(ch)
            close(done)
        }()

        // Read progress messages until channel is closed
        for msg := range ch {
            txtOut = append(txtOut, msg)
        }

        // Wait for executor goroutine to finish
        <-done

        // 3. No error return from executor: handlers must report errors via progress

        // 4. Refresh UI (generic)
        h.tui.RefreshUI()
        
        // 5. Return collected output (generic)
        return mcp.NewToolResultText(strings.Join(txtOut, "\n")), nil
    }
}
```

**This eliminates:**
- ❌ Separate handler functions per tool (`mcpToolWasmSetMode`, `mcpToolWasmRecompile`, etc.)
- ❌ Switch statements mapping tool names to implementations
- ❌ Duplicate code for extracting args and collecting output
- ❌ TinyWasm knowing about specific tool names or logic

**How Reflection Works:**

1. **Call batch method:** `mcpToolsFromHandler` invokes `GetMCPToolsMetadata()` on handler
2. **Extract slice:** Reads `[]ToolMetadata` from any compatible slice type
3. **Convert each tool:** Transforms `handler.ToolMetadata` → `tinywasm.ToolMetadata` field-by-field
4. **Copy Execute function:** Uses reflection to wrap the handler's Execute function
5. **Build MCP tools:** Constructs `mcp.Tool` instances with proper schemas

**Why this works:** Go's type system treats identically-named structs in different packages as incompatible types. Reflection allows us to copy fields AND functions between them.

## Key Files

- `tinywasm/mcp-metadata.go`: Reflection-based metadata converter
- `tinywasm/mcp-executor.go`: Generic tool executor (works for ALL tools)
- `tinywasm/mcp.go`: MCP server setup and automatic tool registration
- `tinywasm/mcp-tool.go`: Handler metadata and execution functions


## Testing Your Tool

1. **Compile and install:**
   ```bash
   cd /home/cesar/Dev/Pkg/Mine/tinywasm
   ./install.sh
   ```

2. **Start TinyWasm:**
   ```bash
   cd example
   tinywasm
   ```

3. **Test via MCP (in another terminal):**
   ```bash
   curl -X POST http://localhost:3030/mcp \
     -H "Content-Type: application/json" \
     -d '{
       "jsonrpc":"2.0",
       "id":1,
       "method":"tools/call",
       "params":{
         "name":"tinywasm-mcp_wasm_set_mode",
         "arguments":{"mode":"L"}
       }
     }'
   ```

4. **Check logs:**
   ```bash
   cat example/logs.log
   ```

## Common Issues

### Method Not Found
**Symptom:** `method GetMCPToolsMetadata not found on handler`

**Cause:** Method name doesn't match exactly or isn't exported.

**Solution:** Method MUST be named exactly `GetMCPToolsMetadata()` and return `[]ToolMetadata`.

### Execute Function Not Working
**Symptom:** Tool registered but execution fails or does nothing

**Cause:** Execute function signature doesn't match `ToolExecutor` type.

**Solution:** Must be `func(args map[string]any, progress chan<- string)`. Report errors by sending messages on `progress`, e.g.:
```go
Execute: func(args map[string]any, progress chan<- string) {
    value, ok := args["param_name"]
    if !ok {
        progress <- "error: missing required parameter 'param_name'"
        return
    }
    
    strValue, ok := value.(string)
    if !ok {
        progress <- "error: parameter must be a string"
        return
    }
    // Use strValue...
    progress <- "done"
}
```

### Type Mismatch Error
**Symptom:** `expected slice, got tinywasm.ToolMetadata`

**Cause:** Returning single `ToolMetadata` instead of slice.

**Solution:** Always return `[]ToolMetadata` (slice), even for single tool:
```go
return []ToolMetadata{
    {Name: "my_tool", Execute: func(...) {...}},
}
```

### Parameter Extraction Fails
**Symptom:** Tool executes but can't read parameters

**Cause:** Parameter name mismatch or wrong type assertion.

**Solution:** See example above for safe extraction and reporting via `progress`.


## Benefits of This Architecture

✅ **Complete Decoupling:** TinyWasm knows NOTHING about tool names or implementations  
✅ **Single Execution Path:** One generic executor handles ALL tools  
✅ **Self-Contained Handlers:** Each tool brings its own metadata + execution logic  
✅ **Zero Boilerplate:** No switch statements, no individual handler functions  
✅ **Type Safe:** Reflection validates structure compatibility  
✅ **Easy to Extend:** Add tool = add element to slice in handler  
✅ **Testable:** Handlers can be tested without MCP server  
✅ **Scalable:** Works for 1 tool or 100 tools identically

## Code Reduction Example

**Before (per-tool handlers):**
```go
// In tinywasm/mcp-wasm.go (~40 lines PER tool)
func (h *handler) mcpToolWasmSetMode(...) { /* extract args, validate, execute, collect output */ }
func (h *handler) mcpToolWasmRecompile(...) { /* extract args, validate, execute, collect output */ }
func (h *handler) mcpToolWasmGetSize(...) { /* extract args, validate, execute, collect output */ }

// In tinywasm/mcp.go (switch per tool)
switch toolMeta.Name {
case "wasm_set_mode": s.AddTool(*tool, h.mcpToolWasmSetMode)
case "wasm_recompile": s.AddTool(*tool, h.mcpToolWasmRecompile)
case "wasm_get_size": s.AddTool(*tool, h.mcpToolWasmGetSize)
}
```

**After (generic executor):**
```go
// In tinywasm/mcp-executor.go (~45 lines total for ALL tools)
func (h *handler) mcpExecuteTool(executor ToolExecutor) { /* generic implementation using a progress channel */ }

// In tinywasm/mcp.go (no switch needed)
for _, toolMeta := range tools {
    s.AddTool(*buildMCPTool(toolMeta), h.mcpExecuteTool(toolMeta.Execute))
}

// In tinywasm/mcp-tool.go (domain logic where it belongs)
Execute: func(args map[string]any, progress chan<- string) {
    mode := args["mode"].(string)
    w.Change(mode, progress)
    progress <- "done"
}
```

**Result:** ~120 lines of repetitive code → ~15 lines of declarative metadata


## Migration Guide

### From Individual Handler Functions

If you have individual tool handler functions:

**Old approach:**
```go
// In tinywasm/mcp-wasm.go
func (h *handler) mcpToolWasmSetMode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args := req.Params.Arguments.(map[string]any)
    mode := args["mode"].(string)
    var output strings.Builder
    progress := func(msg string) { output.WriteString(msg + "\n") }
    h.wasmHandler.Change(mode, progress)
    h.tui.RefreshUI()
    return mcp.NewToolResultText(output.String()), nil
}

// In tinywasm/mcp.go
s.AddTool(mcp.NewTool("wasm_set_mode", ...), h.mcpToolWasmSetMode)
```

**New approach:**
```go
// In tinywasm/mcp-tool.go
func (w *TinyWasm) GetMCPToolsMetadata() []ToolMetadata {
    return []ToolMetadata{
        {
            Name: "wasm_set_mode",
            Description: "...",
            Parameters: []ParameterMetadata{...},
            Execute: func(args map[string]any, progress chan<- string) {
                mode := args["mode"].(string)
                w.Change(mode, progress)
                progress <- "done"
            },
        },
    }
}

// In tinywasm/mcp.go (automatic)
for _, toolMeta := range tools {
    s.AddTool(*buildMCPTool(toolMeta), h.mcpExecuteTool(toolMeta.Execute))
}
```

**Steps:**
1. Move parameter extraction and domain logic to `Execute` function in handler  
2. Remove individual tool handler functions from tinywasm  
3. Remove switch statement in mcp.go registration  
4. Delete mcp-wasm.go (no longer needed)


## Extending to Other Projects

This pattern is framework-agnostic. To use in another project:

1. Copy `mcp-metadata.go` (reflection converter)  
2. Define `ToolMetadata` and `ParameterMetadata` in each domain package  
3. Implement `GetMCPToolsMetadata() []ToolMetadata` method on handlers  
4. Use `mcpToolsFromHandler` in MCP registration layer

No dependencies on TinyWasm-specific code required.
