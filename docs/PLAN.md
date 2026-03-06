# Plan: App Package Restructure + MCP Tool Registry

## References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [MCP_ARCHITECTURE_PLAN.md](MCP_ARCHITECTURE_PLAN.md)
- `bootstrap.go` — current entry point (to be split)
- `start.go` — Handler struct + Start() (to be split)
- `mcp-tools.go` — app.Handler tools
- `devbrowser/mcp-tools.go` — browser tools

---

## Development Rules

- **SRP:** Every file must have a single, well-defined purpose.
- **Max 500 lines per file.** Files exceeding this MUST be subdivided.
- **No global state.** Use dependency injection via interfaces.
- **Standard library only.** No external test assertion libraries.
- **Test runner:** Use `gotest` (never `go test` directly).
- **Publish:** Use `gopush 'message'` after tests pass.
- **Language:** Plans in English, chat in Spanish.
- **No code changes** until the user says "ejecuta" or "ok".

---

## Problem Summary

### 1. `bootstrap.go` violates SRP (476 lines, 4 responsibilities)
- Bootstrap entry point
- Daemon lifecycle (`runDaemon`, `daemonToolProvider`, `startProject`, `runProjectLoop`)
- Network/process helpers (`isPortOpen`, `waitForPort*`, `killDaemon`, `startDaemonProcess`)
- Client mode (`runClient`)

### 2. Tool registration is broken in daemon mode
- `start.go:224-232` registers tools in standalone mode (`h`, `h.Browser`, `h.WasmClient`) — works.
- `bootstrap.go:129` in daemon mode only registers `daemonToolProvider` — browser and app tools are never registered.
- `main.go` always leaves `McpToolHandlers = nil`.

### 3. Lifecycle mismatch (root cause)
The `mcp.Handler` is created **once** at daemon startup, but project-level tool providers (`devbrowser`, `app.Handler`) are created **per project** inside `runProjectLoop`. There is no mechanism to update tool providers after startup.

### 4. `app.Handler.GetMCPTools()` (`app_rebuild`) never reaches the daemon MCP
It is only registered in standalone mode. In daemon mode it is dead code.

---

## Solution: `ProjectToolProxy` + Single Registration Point

Introduce a `ProjectToolProxy` registered **once** with the MCP daemon at startup. When a project starts or stops, `SetActive()` updates the proxy atomically. The MCP handler always reads current tools via `GetMCPTools()`.

```
Daemon startup:
  proxy := NewProjectToolProxy()
  mcp.NewHandler([daemonToolProvider, proxy], ...)

start_development called:
  browser := BrowserFactory(...)
  handler := newProjectHandler(...)
  proxy.SetActive(handler, browser, wasmClient)

stop_development called:
  proxy.SetActive()  // empty
```

All tool registration logic lives exclusively in `mcp_registry.go`. No other file assembles tool provider lists.

---

## File Restructure

### Files to CREATE

#### `net.go`
Extracted from `bootstrap.go`. Contains only TCP/process helpers.
- `isPortOpen(port string) bool`
- `waitForPortFree(port string)`
- `waitForPortReady(port string)`
- `isDaemonVersionCurrent(port, version string) bool`
- `killDaemon()`
- `startDaemonProcess(dir string) error`

#### `daemon.go`
Extracted from `bootstrap.go`. Contains daemon lifecycle only.
- `runDaemon(cfg BootstrapConfig)`
- `daemonToolProvider` struct
- `newDaemonToolProvider(...)`
- `(d *daemonToolProvider) GetMCPTools() []mcp.Tool`
- `(d *daemonToolProvider) startProject(path string)`
- `(d *daemonToolProvider) stopProject()`
- `(d *daemonToolProvider) restartCurrentProject()`
- `(d *daemonToolProvider) runProjectLoop(ctx, path string)`

#### `handler.go`
Extracted from `start.go`. Contains the Handler struct definition only.
- `Handler` struct
- `sseHubAdapter` struct and `Publish` method
- `(h *Handler) SetBrowser(...)`
- `(h *Handler) SetServerFactory(...)`
- `(h *Handler) CheckDevMode()`

#### `mcp_registry.go`
**New file. Single source of truth for all MCP tool registration.**

```go
// ProjectToolProxy is registered once with the MCP daemon at startup.
// When a new project starts, SetActive() updates tool providers atomically.
type ProjectToolProxy struct {
    mu     sync.RWMutex
    active []mcp.ToolProvider
}

func NewProjectToolProxy() *ProjectToolProxy

// SetActive replaces the current project's tool providers.
// Call with no args to clear (project stopped).
func (p *ProjectToolProxy) SetActive(providers ...mcp.ToolProvider)

// GetMCPTools implements mcp.ToolProvider. Always reflects the current project.
func (p *ProjectToolProxy) GetMCPTools() []mcp.Tool

// buildProjectProviders returns the ordered list of tool providers for a given Handler.
// This is the single place where project-level tool registration order is defined.
func buildProjectProviders(h *Handler) []mcp.ToolProvider
```

`buildProjectProviders` implementation:
```go
func buildProjectProviders(h *Handler) []mcp.ToolProvider {
    providers := []mcp.ToolProvider{h} // app_rebuild tool
    if h.WasmClient != nil {
        providers = append(providers, h.WasmClient)
    }
    if h.Browser != nil {
        providers = append(providers, h.Browser)
    }
    return providers
}
```

### Files to MODIFY

#### `bootstrap.go` (after split: ~100 lines)
Keeps only:
- `Bootstrap(cfg BootstrapConfig)` entry point
- `runClient(cfg BootstrapConfig)`
- `logChannelProvider` struct (used in both modes)
- `BootstrapConfig` struct

Removes: everything moved to `daemon.go` and `net.go`.

#### `start.go` (after split: ~180 lines)
Keeps only `Start(...)` function.
Removes: `Handler` struct, `sseHubAdapter`, `SetBrowser`, `SetServerFactory`, `CheckDevMode` (moved to `handler.go`).

Updates tool registration to use `buildProjectProviders`:
```go
// Before (start.go:224-232) — scattered, duplicated logic:
toolHandlers := []mcp.ToolProvider{}
toolHandlers = append(toolHandlers, h)
if h.WasmClient != nil { toolHandlers = append(toolHandlers, h.WasmClient) }
if h.Browser != nil { toolHandlers = append(toolHandlers, h.Browser) }
toolHandlers = append(toolHandlers, mcpToolHandlers...)

// After — single call:
toolHandlers := buildProjectProviders(h)
toolHandlers = append(toolHandlers, mcpToolHandlers...)
```

#### `daemon.go` — `runProjectLoop`
After creating the project handler and browser, update the proxy:
```go
// After Start() initializes h.WasmClient and h.Browser:
proxy.SetActive(buildProjectProviders(h)...)

// On project stop (deferred in runProjectLoop):
defer proxy.SetActive()
```

`runDaemon` passes the proxy as a provider:
```go
proxy := NewProjectToolProxy()
mcpHandler = mcp.NewHandler(
    mcpConfig,
    append(cfg.McpToolHandlers, dtp, proxy),
    ui, sseHub, exitChan,
)
```

#### `main.go` (no structural change needed)
`McpToolHandlers` stays nil — the proxy handles project-level tools. No change required.

---

## Execution Steps

### Step 1 — Create `net.go`
Move all TCP/process helper functions out of `bootstrap.go` into a new `net.go` file. No logic changes.

### Step 2 — Create `handler.go`
Move `Handler` struct, `sseHubAdapter`, `SetBrowser`, `SetServerFactory`, `CheckDevMode` out of `start.go` into `handler.go`. No logic changes.

### Step 3 — Create `daemon.go`
Move `runDaemon`, `daemonToolProvider` and all its methods, `runProjectLoop` out of `bootstrap.go` into `daemon.go`. No logic changes yet.

### Step 4 — Trim `bootstrap.go` and `start.go`
After moves, both files should contain only their core responsibility. Verify line counts < 200.

### Step 5 — Create `mcp_registry.go`
Implement `ProjectToolProxy` and `buildProjectProviders`. Add tests in `test/mcp_registry_test.go`.

### Step 6 — Wire `ProjectToolProxy` in `daemon.go`
Update `runDaemon` to create and register the proxy. Update `runProjectLoop` to call `proxy.SetActive(buildProjectProviders(h)...)` after project start and `defer proxy.SetActive()` on exit.

### Step 7 — Update `start.go` standalone mode
Replace the manual tool list assembly (lines 224-232) with `buildProjectProviders(h)`.

### Step 8 — Run tests and publish
```bash
gotest
gopush 'refactor: split bootstrap, add ProjectToolProxy for unified MCP tool registration'
```

---

## Test Strategy

- `TestProjectToolProxy_EmptyByDefault` — `GetMCPTools()` returns empty slice when no active project.
- `TestProjectToolProxy_SetActive_ExposesBrowserTools` — after `SetActive(browser)`, tools from browser appear.
- `TestProjectToolProxy_SetActive_Clear` — after `SetActive()` with no args, tools return empty.
- `TestProjectToolProxy_ThreadSafe` — concurrent `SetActive` + `GetMCPTools` does not race (run with `-race`).
- `TestBuildProjectProviders_IncludesHandler` — always includes `app.Handler` first.
- `TestBuildProjectProviders_SkipsNilBrowser` — nil browser is not added.
- `TestBuildProjectProviders_SkipsNilWasmClient` — nil WasmClient is not added.

Existing tests must continue passing. No mock changes required (proxy is internal to daemon path).
