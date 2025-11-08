# GoLite: Refactor MCP ToolExecutor to Use Channel-Based Progress

## Objective
Refactor the `ToolExecutor` function signature in GoLite's MCP layer from callback-based `func(msgs ...any)` with error return to channel-based `chan<- string` without error return. This aligns with DevTUI and TinyWasm's new channel-based interfaces.

## Current Signature
```go
// mcp-metadata.go
type ToolExecutor func(args map[string]any, progress func(msgs ...any)) error
```

## Target Signature
```go
// mcp-metadata.go
type ToolExecutor func(args map[string]any, progress chan<- string)
```

## Rationale
1. **Consistency**: DevTUI and TinyWasm now use channels for progress
2. **Simplicity**: Single return path (messages via channel), errors sent as messages
3. **Streaming**: True asynchronous progress updates
4. **Idiomatic Go**: Channels for goroutine communication

## Files to Modify

### 1. `/mcp-metadata.go`
Update ToolExecutor type definition and reflection conversion:

```go
// ToolExecutor defines how a tool should be executed
// Handlers implement this to provide execution logic without exposing internals
// args: map of parameter name to value from MCP request
// progress: channel to send progress messages back to caller
type ToolExecutor func(args map[string]any, progress chan<- string)
```

Update reflection conversion in `convertToToolMetadata` method:

```go
// Extract Execute field (function)
if execField := sourceValue.FieldByName("Execute"); execField.IsValid() && execField.Kind() == reflect.Func {
    // Convert to ToolExecutor by wrapping the function
    meta.Execute = func(args map[string]any, progress chan<- string) {
        // Create reflection values for call
        results := execField.Call([]reflect.Value{
            reflect.ValueOf(args),
            reflect.ValueOf(progress),
        })
        // Note: No error handling - errors sent as messages via channel
    }
}
```

### 2. `/mcp-executor.go`
Update generic executor to use channels:

```go
package golite

import (
	"context"
	"strings"
	
	"github.com/mark3labs/mcp-go/mcp"
)

// mcpExecuteTool creates a GENERIC tool executor that works for ANY handler tool
// It extracts args, collects progress, executes the tool, and returns results
// NO domain-specific logic here - handlers provide their own Execute functions
func (h *handler) mcpExecuteTool(executor ToolExecutor) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// 1. Extract arguments (generic)
		args, ok := req.Params.Arguments.(map[string]any)
		if !ok {
			// Handle case where no arguments are provided
			args = make(map[string]any)
		}

		// 2. Create channel and collect messages
		progressChan := make(chan string, 10) // Buffered to avoid blocking
		messages := []string{}
		done := make(chan bool)

		// 3. Collect progress messages in goroutine
		go func() {
			for msg := range progressChan {
				messages = append(messages, msg)
			}
			done <- true
		}()

		// 4. Execute handler-specific logic (sends messages to channel)
		executor(args, progressChan)
		close(progressChan)

		// 5. Wait for collection to finish
		<-done

		// 6. Refresh UI (generic)
		if h.tui != nil {
			h.tui.RefreshUI()
		}

		// 7. Return collected output (generic)
		if len(messages) == 0 {
			return mcp.NewToolResultText("Operation completed successfully"), nil
		}

		return mcp.NewToolResultText(strings.Join(messages, "\n")), nil
	}
}
```

### 3. `/mcp.go`
No changes needed - already uses reflection-based loading:

```go
// === BUILD CONTROL TOOLS ===

// Load WASM tools from TinyWasm metadata using reflection
if h.wasmHandler != nil {
    if tools, err := mcpToolsFromHandler(h.wasmHandler); err == nil {
        for _, toolMeta := range tools {
            tool := buildMCPTool(toolMeta)
            // Use generic executor - works with new signature automatically
            s.AddTool(*tool, h.mcpExecuteTool(toolMeta.Execute))
        }
    } else {
        h.config.logger("Warning: Failed to load WASM tools:", err)
    }
}
```

### 4. Update Documentation

#### `/docs/MCP-ADD-TOOLS.md`
Update examples to show new signature:

```go
// OLD
Execute: func(args map[string]interface{}, progress func(msgs ...any)) error {
    if err := validateArgs(args); err != nil {
        return err
    }
    progress("Processing...")
    return nil
}

// NEW
Execute: func(args map[string]interface{}, progress chan<- string) {
    if err := validateArgs(args); err != nil {
        progress <- fmt.Sprintf("Error: %v", err)
        return
    }
    progress <- "Processing..."
    progress <- "Completed successfully"
}
```

Update "Step 2" section:
```markdown
### Step 2: Implement GetMCPToolsMetadata Method with Execution Logic

Each tool includes its own `Execute` function with all domain logic:
- `Execute` receives `args` (parameters) and `progress` (channel for messages)
- `Execute` does NOT return error - errors sent as messages via channel
- Close channel handled by generic executor
```

Update "Common Issues" section:
```markdown
### Execute Function Not Working
**Symptom:** Tool registered but execution fails or does nothing

**Cause:** Execute function signature doesn't match `ToolExecutor` type.

**Solution:** Must be `func(args map[string]any, progress chan<- string)` (no error return)

### Error Messages
**Symptom:** How to handle errors without returning error?

**Solution:** Send error messages via channel:
```go
Execute: func(args map[string]any, progress chan<- string) {
    if err := someOperation(); err != nil {
        progress <- fmt.Sprintf("Error: %v", err)
        return
    }
    progress <- "Success"
}
```

## Implementation Steps

1. **Update mcp-metadata.go** - Change ToolExecutor signature and reflection conversion
2. **Update mcp-executor.go** - Refactor to use channels instead of callbacks
3. **Update docs/MCP-ADD-TOOLS.md** - Update examples and troubleshooting
4. **Test compilation** - Ensure no errors: `go build ./...`
5. **Test MCP server** - Start golite and verify tools work

## Key Implementation Details

### Channel Management
- **Buffer size**: 10 (handles multiple progress messages)
- **Closing**: Generic executor closes channel after tool execution
- **Collection**: Separate goroutine collects all messages
- **Blocking**: Buffered channel prevents blocking tool execution

### Error Handling Pattern
```go
// OLD (with error return)
Execute: func(args map[string]any, progress func(msgs ...any)) error {
    if err := validate(args); err != nil {
        return err // Stops execution
    }
    progress("Success")
    return nil
}

// NEW (channel-based)
Execute: func(args map[string]any, progress chan<- string) {
    if err := validate(args); err != nil {
        progress <- fmt.Sprintf("Error: %v", err) // Send error message
        return // Stop execution
    }
    progress <- "Success"
}
```

### Reflection Conversion
The reflection code in `convertToToolMetadata` automatically wraps the handler's Execute function:
- Accepts handler function with signature `func(map[string]any, chan<- string)`
- Wraps it to match GoLite's `ToolExecutor` type
- Handles type differences between packages (e.g., `tinywasm.ToolExecutor` → `golite.ToolExecutor`)

## Breaking Changes
⚠️ **This is a BREAKING CHANGE** for any external handlers implementing MCP tools.

**Affected:**
- TinyWasm (will be updated separately)
- DevBrowser (if implementing MCP tools)
- Any other handlers exposing MCP tools

## Success Criteria
- [ ] ToolExecutor signature updated to use channel
- [ ] mcp-executor.go refactored to use channels
- [ ] Reflection conversion updated
- [ ] Documentation updated
- [ ] Code compiles: `go build ./...`
- [ ] MCP server starts without errors
- [ ] Tools respond correctly via MCP

## Test Verification

### Compilation Test
```bash
cd /home/cesar/Dev/Pkg/Mine/golite
go build ./...
```

### Runtime Test
```bash
# Terminal 1: Start golite
cd example
golite

# Terminal 2: Test MCP tool (after TinyWasm updated)
curl -X POST http://localhost:3030/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "id":1,
    "method":"tools/call",
    "params":{
      "name":"golite-mcp_wasm_set_mode",
      "arguments":{"mode":"L"}
    }
  }'
```

## Dependencies
This refactor must be done AFTER:
1. ✅ DevTUI updated to use `chan<- string`
2. ✅ TinyWasm updated to use `chan<- string`

Then GoLite can be updated to match the new interface.

## Notes
- No test files need updating (mcp_test.go tests tool stubs, not Execute functions)
- Generic executor handles all channel management
- Handlers don't need to know about buffering or closing
- Error messages formatted before sending to channel
- Multiple messages allowed per tool execution
