# GoLite MCP Server Tools

**Model Context Protocol (MCP) Server for GoLite Framework**

## Overview

GoLite MCP server provides LLM agents with programmatic control over the GoLite development environment. This enables AI assistants to monitor build status, control compilation modes, manage the development browser, and debug WebAssembly applications during full-stack Go development.

**Purpose**: Allow LLMs to interact with GoLite as a development tool, not to edit user project files directly.

---

## Tool Categories

### 1. Status & Monitoring Tools
Tools for observing current state and logs.

### 2. Build Control Tools
Tools for compilation and WASM mode management.

### 3. Browser Control Tools
Tools for managing the development browser.

### 4. Deployment Tools
Tools for Cloudflare Workers deployment status.

---

## Tool Specifications

### 🔍 `golite_status`
**Get comprehensive status of GoLite environment**

Returns current state of all components: server, WASM compiler, assets, watcher, and browser.

**Arguments**: None

**Returns**:
```json
{
  "framework": "GOLITE",
  "root_dir": "/path/to/project",
  "server": {
    "running": true,
    "port": "4430",
    "output_dir": "deploy/appserver"
  },
  "wasm": {
    "mode": "MEDIUM",
    "compiler": "tinygo",
    "output_dir": "src/web/public",
    "available_modes": ["LARGE", "MEDIUM", "SMALL"]
  },
  "browser": {
    "open": true,
    "url": "http://localhost:4430"
  },
  "assets": {
    "watching": true,
    "public_dir": "src/web/public"
  }
}
```

**Justification**: Essential for LLM to understand current environment state before taking actions. Prevents blind operations and enables context-aware debugging.

---

### 📝 `golite_get_logs`
**Retrieve recent logs from specific component**

Fetches last N log entries from BUILD or DEPLOY sections.

**Arguments**:
- `component` (required): `"WASM"` | `"SERVER"` | `"ASSETS"` | `"WATCH"` | `"BROWSER"` | `"CLOUDFLARE"`
- `lines` (optional, default: 50): Number of recent log lines to retrieve

**Returns**:
```json
{
  "component": "WASM",
  "logs": [
    "[15:04:05] Compiling main.wasm with tinygo...",
    "[15:04:08] ✓ Compilation successful (2.3MB)",
    "[15:04:08] Browser reload triggered"
  ],
  "timestamp": "2025-11-05T15:04:08Z"
}
```

**Justification**: Critical for debugging. LLMs need to see compilation errors, warnings, and build output to diagnose issues without accessing TUI.

---

### 🔧 `wasm_set_mode`
**Change WASM compilation mode**

Switches between LARGE (Go std), MEDIUM (TinyGo optimized), or SMALL (TinyGo ultra-compact).

**Arguments**:
- `mode` (required): `"LARGE"` | `"L"` | `"MEDIUM"` | `"M"` | `"SMALL"` | `"S"`

**Returns**:
```json
{
  "previous_mode": "MEDIUM",
  "new_mode": "SMALL",
  "compiler": "tinygo",
  "auto_recompile": true,
  "message": "Mode changed to Small (tinygo). Auto-recompiling..."
}
```

**Justification**: Essential for optimization workflow. LLMs can help users reduce WASM size by testing different modes and measuring results. Common debugging pattern: "Try LARGE mode if TinyGo has issues."

---

### 🔄 `wasm_recompile`
**Force WASM recompilation**

Triggers immediate recompilation of main WASM file with current mode.

**Arguments**: None

**Returns**:
```json
{
  "mode": "MEDIUM",
  "compiler": "tinygo",
  "status": "success",
  "output_size": "1.8MB",
  "duration": "3.2s",
  "browser_reloaded": true
}
```

**Justification**: Sometimes automatic rebuild fails or user edits weren't detected. LLM can force rebuild when asked "why isn't my change showing up?"

---

### 🌐 `browser_open`
**Open development browser**

Launches Chrome in development mode pointing to local server.

**Arguments**: None

**Returns**:
```json
{
  "status": "opened",
  "url": "http://localhost:4430",
  "message": "Browser opened and navigated to application"
}
```

**Justification**: LLM can help user start browser if they closed it or it crashed. Common request: "open the browser again."

---

### 🌐 `browser_close`
**Close development browser**

Closes the controlled browser instance and cleans up resources.

**Arguments**: None

**Returns**:
```json
{
  "status": "closed",
  "message": "Browser closed successfully"
}
```

**Justification**: Cleanup operation. User might say "close browser, I want to test manually" or browser might be stuck.

---

### 🌐 `browser_reload`
**Reload browser page**

Forces browser page refresh without full browser restart.

**Arguments**: None

**Returns**:
```json
{
  "status": "reloaded",
  "url": "http://localhost:4430",
  "timestamp": "2025-11-05T15:04:08Z"
}
```

**Justification**: Quick refresh for testing changes. Faster than close/open cycle. Common in debugging: "reload to see if error persists."

---

### 🌐 `browser_get_console`
**Get browser console logs**

Retrieves JavaScript console output from browser (errors, warnings, logs).

**Arguments**:
- `level` (optional, default: "all"): `"all"` | `"error"` | `"warning"` | `"log"`
- `lines` (optional, default: 50): Number of recent entries

**Returns**:
```json
{
  "console_logs": [
    {"level": "error", "message": "Uncaught TypeError: Cannot read property 'value'", "timestamp": "15:04:05"},
    {"level": "log", "message": "WASM initialized", "timestamp": "15:04:03"},
    {"level": "warning", "message": "Slow network detected", "timestamp": "15:04:01"}
  ],
  "filtered_by": "all",
  "count": 3
}
```

**Justification**: **CRITICAL** for debugging frontend issues. Most user problems manifest as browser console errors. LLM needs this to diagnose "app not working" complaints without user manually copying errors.

---

### 📊 `wasm_get_size`
**Get current WASM file size**

Returns size of compiled WASM file and comparison across modes.

**Arguments**: None

**Returns**:
```json
{
  "current_mode": "MEDIUM",
  "current_size": "1.8MB",
  "sizes_by_mode": {
    "LARGE": "5.2MB",
    "MEDIUM": "1.8MB", 
    "SMALL": "892KB"
  },
  "recommendation": "SMALL mode could reduce size by 50%"
}
```

**Justification**: Users constantly ask "why is my WASM so big?" LLM can analyze and suggest appropriate mode based on size/features tradeoff.

---

### 🚀 `deploy_status`
**Get Cloudflare deployment status**

Returns current deployment configuration and last deploy result.

**Arguments**: None

**Returns**:
```json
{
  "configured": true,
  "input_dir": "src/cmd/edgeworker",
  "output_dir": "deploy/edgeworker",
  "last_compile": {
    "status": "success",
    "timestamp": "2025-11-05T14:30:00Z",
    "output_file": "_worker.js"
  }
}
```

**Justification**: LLM can confirm if deployment pipeline is set up correctly. Useful when user asks "how do I deploy this?"

---

### 📂 `project_structure`
**Get project directory structure**

Returns conventional GoLite directory structure with file counts.

**Arguments**: None

**Returns**:
```json
{
  "root": "/path/to/project",
  "structure": {
    "src/cmd/appserver": {"files": 3, "purpose": "Backend Go server"},
    "src/cmd/webclient": {"files": 5, "purpose": "Frontend WASM entry point"},
    "src/web/ui": {"files": 12, "purpose": "HTML/CSS/JS themes"},
    "src/web/public": {"files": 4, "purpose": "Generated assets"},
    "deploy/appserver": {"files": 1, "purpose": "Compiled backend binary"},
    "deploy/edgeworker": {"files": 1, "purpose": "Cloudflare Worker script"}
  },
  "valid": true
}
```

**Justification**: Helps LLM understand project layout to give accurate advice. Can detect missing directories or wrong structure causing errors.

---

### 🔍 `check_requirements`
**Verify development environment**

Checks if required tools (Go, TinyGo, Chrome) are installed.

**Arguments**: None

**Returns**:
```json
{
  "go": {
    "installed": true,
    "version": "1.21.0",
    "path": "/usr/local/go/bin/go"
  },
  "tinygo": {
    "installed": true,
    "version": "0.30.0",
    "path": "/usr/local/bin/tinygo"
  },
  "chrome": {
    "installed": true,
    "path": "/usr/bin/google-chrome"
  },
  "all_ready": true
}
```

**Justification**: First diagnostic step. If TinyGo missing, LLM can explain why MEDIUM/SMALL modes don't work and guide installation.

---

## Implementation Notes

### Server Integration

The MCP server should be started automatically when `golite` runs, similar to how other components are initialized in `Start()`:

```go
// In start.go
func Start(rootDir string, logger func(messages ...any), ui TuiInterface, exitChan chan bool) {
    h := &handler{
        frameworkName: "GOLITE",
        rootDir:       rootDir,
        tui:           ui,
        exitChan:      exitChan,
    }
    
    // ... existing initialization ...
    
    // Start MCP server
    go h.ServeMCP()
    
    // ... rest of initialization ...
}
```

### Tool Handler Pattern

Each tool follows this implementation pattern:

```go
func (h *handler) mcpToolGetStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    status := map[string]interface{}{
        "framework": h.frameworkName,
        "root_dir": h.rootDir,
        "server": map[string]interface{}{
            "running": h.serverHandler != nil,
            "port": h.config.ServerPort(),
        },
        // ... collect status from all components
    }
    
    jsonData, _ := json.Marshal(status)
    return mcp.NewToolResultText(string(jsonData)), nil
}
```

### Browser Console Integration

Requires extending `devbrowser` package to capture console logs via Chrome DevTools Protocol (CDP):

```go
// In devbrowser package
func (b *DevBrowser) GetConsoleLogs(level string, lines int) []ConsoleLog {
    // Use rod's page.EvalOnNewDocument or page.Evaluate
    // to capture console.log, console.error, console.warn
}
```

### Log Buffer Management

Each logger should maintain a circular buffer of recent messages:

```go
type LogBuffer struct {
    entries []LogEntry
    maxSize int
    mu      sync.RWMutex
}

func (lb *LogBuffer) Add(message string) {
    lb.mu.Lock()
    defer lb.mu.Unlock()
    // Add with timestamp, trim if exceeds maxSize
}
```

---

## Usage Examples

### Debugging Scenario

**User**: "My app shows blank page"

**LLM Actions**:
1. `golite_status` → Check if server and browser running
2. `browser_get_console` → Check for JS errors
3. `golite_get_logs component=WASM` → Check WASM compilation
4. Analyze errors and suggest fix

### Optimization Scenario  

**User**: "WASM file is too large"

**LLM Actions**:
1. `wasm_get_size` → Current: 5.2MB (LARGE mode)
2. `wasm_set_mode mode=SMALL` → Switch to TinyGo compact
3. `wasm_get_size` → Now: 892KB
4. `browser_reload` → Test if app still works
5. `browser_get_console` → Verify no new errors

### Setup Verification

**User**: "Is my environment ready?"

**LLM Actions**:
1. `check_requirements` → Verify tools installed
2. `project_structure` → Validate directory layout
3. `golite_status` → Confirm all services running
4. Report any issues found

---

## Design Rationale

### Why These Tools?

**Minimal but Complete**: 13 tools cover all critical operations without overwhelming the LLM with options.

**Observation + Action Balance**:
- 5 status/monitoring tools (read-only, safe)
- 5 control tools (actions with clear effects)
- 3 environment tools (setup verification)

**No File Editing**: GoLite is a build tool, not an editor. LLM controls the tool, doesn't modify user code directly.

**Browser Focus**: 4 browser-related tools because frontend debugging is the most common pain point in WASM development.

### What's NOT Included?

- **File system operations**: Use native MCP file tools or IDE
- **Code generation**: Not GoLite's responsibility
- **Git operations**: Use native MCP git tools
- **Server restart**: Auto-handled by file watcher
- **Asset minification**: Auto-handled, no manual control needed

### Tool Granularity

Tools are deliberately coarse-grained. Example: `wasm_set_mode` both changes mode AND recompiles. This prevents LLMs from getting stuck in "change mode, forget to recompile" patterns.

---

## Security Considerations

### Safe by Design

All tools operate on the **running GoLite instance**, not arbitrary file system paths. The tool can only:
- Read status of current session
- Control current development environment
- View logs from current process

### No Destructive Operations

No tool can:
- Delete user files
- Modify source code
- Execute arbitrary commands
- Access files outside project root

### Rate Limiting

Recommended limits:
- `browser_get_console`: Max 10 calls/minute (avoid log spam)
- `wasm_recompile`: Max 5 calls/minute (compilation is expensive)
- Status tools: Unlimited (read-only, cheap)

---

## Extension Points

Future tools could include:

- `server_get_logs`: Backend server HTTP request logs
- `wasm_benchmark`: Run performance tests on WASM code
- `assets_stats`: Asset bundle size breakdown
- `deploy_trigger`: Initiate Cloudflare deployment
- `browser_screenshot`: Capture current page state
- `browser_network`: Show network requests/responses

But **start with these 13 core tools first**. Add more only if users frequently request specific operations.

---

## Conclusion

These 13 tools give LLMs complete visibility and control over the GoLite development workflow. The selection prioritizes:

1. **Debugging** (logs, console, status)
2. **Optimization** (WASM modes, sizes)
3. **Development flow** (browser control, compilation)

This minimal set avoids overwhelming the LLM while covering 95% of real-world development and troubleshooting scenarios.
