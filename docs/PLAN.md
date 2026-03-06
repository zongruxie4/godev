# Plan: App — Own the HTTP Server + Fix Action Routing

## References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- `bootstrap.go` — daemon entry point
- `daemon.go` — project lifecycle
- `start.go` — standalone MCP setup (to be updated)
- `handler.go` — Handler struct (to be updated)
- `mcp_registry.go` — ProjectToolProxy
- [tinywasm/mcp PLAN.md](../../mcp/docs/PLAN.md) — mcp becomes pure JSON-RPC handler (prerequisite)

---

## Development Rules

- **SRP:** Every file must have a single, well-defined purpose.
- **Max 500 lines per file.**
- **No global state.** Use DI via interfaces.
- **Test runner:** `gotest`. **Publish:** `gopush`.
- **Language:** Plans in English, chat in Spanish.
- **No code changes** until the user says "ejecuta" or "ok".

---

## Problem Summary

After `tinywasm/mcp` removes its HTTP server, `/action`, `/state`, `/version` and SSE
endpoints, `tinywasm/app` must own the full HTTP server lifecycle as the orchestrator.

Additionally:
- `PublishTabLog` / `PublishLog` move from mcp to app (they contain business logic:
  hardcoded tab names, handler names, colors)
- `LogEntry` struct moves from `mcp/handler.go` to `app/http_server.go` — the JSON
  field names (`tab_title`, `handler_name`, etc.) must remain identical so `devtui/sse_client.go`
  can still deserialize without changes
- `h.MCP *mcp.Handler` field is removed from `Handler` struct; replaced by `h.HTTP *TinywasmHTTP`
  (which implements `Name()` + `SetLog()` for TUI registration)
- `BootstrapConfig` needs an `AppName string` field (currently hardcoded as `"tinywasm"` in mcpConfig)
- The `restart` action key changes from `"r"` to `"restart"` — ensure devtui PLAN is consistent
  (currently no client sends `restart`, so it remains a daemon-internal action only)

---

## Solution: `app/http_server.go` — The Tinywasm HTTP Server

This file owns:
1. Full `http.Server` lifecycle
2. Route registration: `/mcp`, `/logs`, `/action`, `/state`, `/version`
3. `PublishTabLog` / `PublishLog` (SSE publishing with business context)
4. `LogEntry` struct (moved from `mcp/handler.go` — JSON field names unchanged)
5. Action dispatch (`/action` → `onAction` callback)
6. State provider (`/state` → `onState` callback)

```go
// TinywasmHTTP is the HTTP server for the TinyWASM daemon and standalone mode.
// It owns the http.Server, SSE hub, and all tinywasm-specific routes.
// Implements Name() + SetLog() so it can be registered as a TUI handler.
type TinywasmHTTP struct {
    port       string
    sseHub     ssePublisher  // interface: Publish(data, channel) + http.Handler
    mcpHTTP    http.Handler  // from mcp.Handler.HTTPHandler()
    onAction   func(key, value string)
    onState    func() []byte
    appVersion string
    log        func(messages ...any)
    server     *http.Server
    mu         sync.Mutex
}

// NewTinywasmHTTP creates the HTTP server with all routes pre-configured.
func NewTinywasmHTTP(port string, mcpHTTP http.Handler, sseHub ssePublisher, appVersion string) *TinywasmHTTP

// OnAction registers the callback for POST /action?key=...
func (s *TinywasmHTTP) OnAction(fn func(key, value string))

// OnState registers the callback for GET /state
func (s *TinywasmHTTP) OnState(fn func() []byte)

// SetLog satisfies the Loggable interface — app registers TinywasmHTTP in the TUI.
func (s *TinywasmHTTP) SetLog(fn func(messages ...any))

// Name satisfies the Loggable interface — returns "MCP" for TUI display.
func (s *TinywasmHTTP) Name() string { return "MCP" }

// Serve starts the HTTP server and blocks until exitChan is closed.
func (s *TinywasmHTTP) Serve(exitChan chan bool)

// Stop gracefully shuts down the HTTP server.
func (s *TinywasmHTTP) Stop()

// PublishTabLog publishes a structured log entry to the SSE stream.
func (s *TinywasmHTTP) PublishTabLog(tabTitle, handlerName, handlerColor, msg string)

// PublishLog publishes a plain log to the BUILD tab under "MCP" handler.
func (s *TinywasmHTTP) PublishLog(msg string)
```

### SSE publisher interface (internal to app)
```go
// ssePublisher is the DI interface for SSE transport.
// Implemented by tinywasm/sse.SSEServer.
type ssePublisher interface {
    http.Handler
    Publish(data []byte, channel string)
}
```

### `LogEntry` struct (moved from mcp, JSON field names unchanged)
```go
// LogEntry is the wire format for SSE log events consumed by devtui/sse_client.go.
// JSON field names MUST match tabContentDTO in devtui — do not rename.
type LogEntry struct {
    Id           string `json:"id"`
    Timestamp    string `json:"timestamp"`
    Content      string `json:"content"`
    Type         uint8  `json:"type"`
    TabTitle     string `json:"tab_title"`
    HandlerName  string `json:"handler_name"`
    HandlerColor string `json:"handler_color"`
    HandlerType  int    `json:"handler_type"`
}
```

---

## Files to Create/Modify

### CREATE: `app/http_server.go`
Contains: `TinywasmHTTP`, `ssePublisher`, `LogEntry`, `PublishTabLog`, `PublishLog`.
`Name()` returns `"MCP"` so existing TUI display labels stay the same.

### MODIFY: `bootstrap.go` — add `AppName` to `BootstrapConfig`
```go
type BootstrapConfig struct {
    AppName string // e.g. "tinywasm" — used in HTTP server version endpoint
    // ... existing fields unchanged
}
```

### MODIFY: `daemon.go` — replace `mcp.Handler` full setup with `TinywasmHTTP`

Before:
```go
mcpHandler = mcp.NewHandler(mcpConfig, toolProviders, ui, sseHub, exitChan)
mcpHandler.SetLog(logger)
mcpHandler.ConfigureIDEs()
mcpHandler.OnUIAction(func(key, value string) { ... })
mcpHandler.RegisterStateProvider(...)
mcpHandler.Serve()
```

After:
```go
mcpHandler := mcp.NewHandler(mcpConfig, toolProviders)
mcpHandler.SetLog(logger)
mcpHandler.ConfigureIDEs()

httpSrv := NewTinywasmHTTP(mcpPort, mcpHandler.HTTPHandler(), sseHub, cfg.Version)
httpSrv.SetLog(logger)
httpSrv.OnAction(func(key, value string) {
    if ui.DispatchAction(key, value) {
        return
    }
    switch key {
    case "stop":
        logger("Stop command received from UI")
        dtp.stopProject()
    case "restart":
        logger("Restart command received from UI")
        dtp.restartCurrentProject()
    default:
        logger("Unknown UI action:", key)
    }
})
httpSrv.OnState(func() []byte { return ui.GetHandlerStates() })
dtp.httpSrv = httpSrv
go httpSrv.Serve(exitChan)
```

Note: `"restart"` is kept as a daemon-internal action (no client sends it yet, but
`restartCurrentProject` is still valid for future use).

Update `runProjectLoop` relay log wiring:
```go
// Before:
headlessTui.RelayLog = func(tabTitle, handlerName, color, msg string) {
    d.mcpHandler.PublishTabLog(tabTitle, handlerName, color, msg)
}

// After:
headlessTui.RelayLog = func(tabTitle, handlerName, color, msg string) {
    d.httpSrv.PublishTabLog(tabTitle, handlerName, color, msg)
}
```

Add `httpSrv *TinywasmHTTP` field to `daemonToolProvider` (remove `mcpHandler *mcp.Handler`).

### MODIFY: `start.go` — standalone mode (non-headless path)

Before:
```go
toolHandlers := buildProjectProviders(h)
toolHandlers = append(toolHandlers, mcpToolHandlers...)
h.MCP = mcp.NewHandler(mcpConfig, toolHandlers, h.Tui, sseHub, h.ExitChan)
h.Tui.AddHandler(h.MCP, colorOrangeLight, h.SectionBuild)
h.MCP.ConfigureIDEs()
SetActiveHandler(h)
go h.MCP.Serve()
```

After:
```go
toolHandlers := buildProjectProviders(h)
toolHandlers = append(toolHandlers, mcpToolHandlers...)

mcpHandler := mcp.NewHandler(mcpConfig, toolHandlers)
mcpHandler.SetLog(h.Logger)
mcpHandler.ConfigureIDEs()

h.HTTP = NewTinywasmHTTP(mcpPort, mcpHandler.HTTPHandler(), sseHub, "")
h.HTTP.SetLog(h.Logger)
h.HTTP.OnState(func() []byte { return h.Tui.GetHandlerStates() })
// No OnAction in standalone — all handlers dispatch locally via TUI

h.Tui.AddHandler(h.HTTP, colorOrangeLight, h.SectionBuild)
SetActiveHandler(h)
go h.HTTP.Serve(h.ExitChan)
```

Update logger wrapper (currently uses `h.MCP.PublishLog`):
```go
// Before:
h.Logger = func(messages ...any) {
    logger(messages...)
    if h.MCP != nil {
        h.MCP.PublishLog(fmt.Sprint(messages...))
    }
}

// After:
h.Logger = func(messages ...any) {
    logger(messages...)
    if h.HTTP != nil {
        h.HTTP.PublishLog(fmt.Sprint(messages...))
    }
}
```

### MODIFY: `handler.go` — update Handler struct

```go
// Before:
MCP *mcp.Handler

// After:
HTTP *TinywasmHTTP
```

Remove `mcp` import if no longer used elsewhere in `handler.go`.

---

## Execution Steps

### Step 1 — Apply `tinywasm/mcp` changes first (prerequisite)
`mcp.Handler.HTTPHandler()` must exist and `NewHandler` must have the new signature.

### Step 2 — Create `app/http_server.go`
Implement `TinywasmHTTP`, `ssePublisher`, `LogEntry`, `PublishTabLog`, `PublishLog`.
`Name()` returns `"MCP"`. `SetLog()` stores the logger.

### Step 3 — Update `bootstrap.go`
Add `AppName string` to `BootstrapConfig`.
Update `main.go` to pass `AppName: "tinywasm"`.

### Step 4 — Update `daemon.go`
- Replace `mcp.NewHandler(...)` with new pattern using `TinywasmHTTP`
- Replace `daemonToolProvider.mcpHandler` field with `httpSrv *TinywasmHTTP`
- Update `runProjectLoop` relay log wiring
- Change `case "q"` → `case "stop"` in action switch

### Step 5 — Update `start.go`
Replace `h.MCP` with `h.HTTP`. Update logger wrapper. Update `AddHandler` call.

### Step 6 — Update `handler.go`
Replace `MCP *mcp.Handler` with `HTTP *TinywasmHTTP`.

### Step 7 — Run tests and publish
```bash
gotest
gopush 'feat: app owns HTTP server, PublishTabLog, action/state routes'
```

---

## Test Strategy

- `TestTinywasmHTTP_ActionRoute` — `POST /action?key=stop` → `onAction` called with `"stop"`
- `TestTinywasmHTTP_StateRoute` — `GET /state` → `onState()` result returned as JSON
- `TestTinywasmHTTP_VersionRoute` — `GET /version` → JSON `{"version":"..."}` with correct appVersion
- `TestPublishTabLog_JSONFieldNames` — published JSON has fields `tab_title`, `handler_name`,
  `handler_color`, `handler_type` — matching `devtui.tabContentDTO` exactly
- `TestTinywasmHTTP_Name` — `Name()` returns `"MCP"`
- Existing `test/mcp_test.go` and server tests must continue passing
