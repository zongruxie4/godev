# Plan: tinywasm/app — Revert TinywasmHTTP + Fix ProjectToolProxy

← Requires: [mcp PLAN](../../mcp/docs/PLAN.md) executed first
→ After this: [devtui PLAN](../../devtui/docs/PLAN.md)

## References
- [ARCHITECTURE.md](ARCHITECTURE.md)
- `app/http_server.go` — TinywasmHTTP (DELETE)
- `app/daemon.go` — daemon tool provider + runDaemon
- `app/start.go` — standalone mode MCP setup
- `app/handler.go` — Handler struct
- `app/mcp_registry.go` — ProjectToolProxy
- `app/bootstrap.go` — Bootstrap entry point, runClient, TuiFactory signature

---

## Development Rules
- **SRP:** Every file must have a single, well-defined purpose.
- **Max 500 lines per file.**
- **No global state.** Use DI via interfaces.
- **Standard library only** in test assertions.
- **Test runner:** `gotest`. **Publish:** `gopush`.
- **Language:** Plans in English, chat in Spanish.
- **No code changes** until the user says "ejecuta" or "ok".
- **No tinywasm/user dependency** — `tinywasm/app` does not use `tinywasm/user`.

---

## Problem Summary

`app/http_server.go` (`TinywasmHTTP`) is a second HTTP server that must not exist.
Three bugs need fixing:

1. **Bug 1 (daemon.go)** — `ProjectToolProxy` created AFTER `HTTPHandler()` call →
   proxy never included in MCPServer's tool registry
2. **Bug 2 (mcp/handler.go)** — `HTTPHandler()` snapshotted tools at call time →
   MCPServer static, proxy's dynamic tools invisible forever
3. **Bug 3 (bootstrap.go:runClient)** — `http.Post(baseURL + "/action?key=start")` is
   a plain REST call to a route that no longer exists after mcp PLAN executes; must
   become a JSON-RPC call via `mcp.Client`

This plan:
- **Deletes** `app/http_server.go`
- **Restores** `mcp.Handler` as sole HTTP server (mcp PLAN prerequisite)
- **Creates** `app/sse_publisher.go` — SSE publishing logic only (no HTTP)
- **Creates** `app/api_key.go` — API key generation and persistence logic
- **Fixes** all three bugs

---

## API Key Architecture

The daemon generates a random API key at startup to secure its MCP endpoint.
This key is shared with the devtui client (running in a separate process) so it
can authenticate its requests.

**Key lifecycle:**
1. Daemon generates key on first start via `generateAPIKey()`
2. Daemon persists key to a file at `cfg.APIKeyPath` (provided by `BootstrapConfig`)
3. Daemon calls `mcpHandler.SetAuth(mcp.NewTokenAuthorizer(key))`
4. Daemon calls `mcpHandler.SetAPIKey(key)` → written into IDE config headers
5. `runClient` (separate process) reads key from `cfg.APIKeyPath`
6. `runClient` passes key to `TuiFactory` and to `mcp.NewClient` for JSON-RPC calls

**If `cfg.APIKeyPath` is empty:** no key is generated or persisted; daemon runs open
(`noopAuthorizer` default in mcp.Handler). Suitable for local/trusted environments.

---

## New `BootstrapConfig` fields

```go
type BootstrapConfig struct {
    StartDir        string
    McpMode         bool
    Debug           bool
    Version         string
    AppName         string
    APIKeyPath      string // path to persist API key; empty = no auth (open mode)
    Logger          func(messages ...any)
    DB              DB
    GitHandler      devflow.GitClient
    GoModHandler    devflow.GoModInterface
    ServerFactory   ServerFactory
    // APIKey added: app passes key to TUI so devtui can auth its /mcp + /logs requests
    TuiFactory      func(exitChan chan bool, clientMode bool, clientURL, apiKey string) TuiInterface
    BrowserFactory  func(ui TuiInterface, exitChan chan bool) BrowserInterface
    GitHubAuth      any
    McpToolHandlers []mcp.ToolProvider
}
```

---

## CREATE: `app/api_key.go`

Single responsibility: API key generation and file-based persistence.

```go
package app

import (
    "crypto/rand"
    "encoding/hex"
    "errors"
    "os"
)

// generateAPIKey generates a cryptographically secure random 32-byte hex key.
func generateAPIKey() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}

// loadOrCreateAPIKey reads the key from path; generates and persists if absent.
// Returns ("", nil) when path is empty — caller treats this as open/no-auth mode.
func loadOrCreateAPIKey(path string) (string, error) {
    if path == "" {
        return "", nil
    }
    data, err := os.ReadFile(path)
    if err == nil {
        return string(data), nil
    }
    if !errors.Is(err, os.ErrNotExist) {
        return "", err
    }
    key, err := generateAPIKey()
    if err != nil {
        return "", err
    }
    return key, os.WriteFile(path, []byte(key), 0600)
}

// readAPIKey reads the key from path without generating.
// Used by runClient (separate process, daemon already created the key).
func readAPIKey(path string) string {
    if path == "" {
        return ""
    }
    data, _ := os.ReadFile(path)
    return string(data)
}
```

---

## CREATE: `app/sse_publisher.go`

Tinywasm-specific SSE publishing: hardcoded tab names, colors, HandlerTypeLoggable.
Not an HTTP server — only constructs and sends SSE events via injected hub.

```go
package app

import (
    "encoding/json"
    "fmt"
    "time"
)

// ssePublisher is the DI interface for SSE transport (tinywasm/sse.SSEServer).
// Single method — all message types (logs + refresh signals) go through Publish.
// Signatures match tinywasm/sse after the variadic Publish fix (see mcp PLAN).
type ssePublisher interface {
    Publish(data []byte, channels ...string)
}

// LogEntry is the SSE wire format consumed by devtui/sse_client.go.
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

// SSEPublisher wraps an ssePublisher hub with tinywasm-specific publishing logic.
// This is NOT an HTTP server — only handles SSE message construction.
type SSEPublisher struct{ hub ssePublisher }

func NewSSEPublisher(hub ssePublisher) *SSEPublisher { return &SSEPublisher{hub: hub} }

func (p *SSEPublisher) PublishTabLog(tabTitle, handlerName, handlerColor, msg string) {
    if p.hub == nil { return }
    entry := LogEntry{
        Id:           fmt.Sprintf("%d", time.Now().UnixNano()),
        Timestamp:    time.Now().Format("15:04:05"),
        Content:      msg, Type: 1,
        TabTitle:     tabTitle,
        HandlerName:  handlerName,
        HandlerColor: handlerColor,
        HandlerType:  4, // HandlerTypeLoggable
    }
    data, _ := json.Marshal(entry)
    p.hub.Publish(data, "logs") // variadic: passes single channel "logs"
}

func (p *SSEPublisher) PublishLog(msg string) {
    p.PublishTabLog("BUILD", "MCP", "#f97316", msg)
}

// PublishStateRefresh sends a lightweight signal to connected devtui clients
// telling them to re-fetch handler state via the tinywasm/state JSON-RPC call.
// Does NOT carry state payload — devtui always pulls state from JSON-RPC.
// Uses reserved HandlerType=0 as the refresh signal marker; devtui checks for it.
func (p *SSEPublisher) PublishStateRefresh() {
    if p.hub == nil { return }
    signal, _ := json.Marshal(map[string]any{"handler_type": 0}) // TypeStateRefresh
    p.hub.Publish(signal, "logs")
}
```

---

## MODIFY: `app/handler.go`

```go
// Before:
HTTP *TinywasmHTTP

// After:
MCP *mcp.Handler
```

---

## MODIFY: `app/bootstrap.go`

### `TuiFactory` signature

`TuiFactory` receives `apiKey` — the client process reads the key from
`cfg.APIKeyPath` and passes it through so devtui can auth both SSE and `/mcp`
requests.

```go
// Before:
TuiFactory func(exitChan chan bool, clientMode bool, clientURL string) TuiInterface

// After:
TuiFactory func(exitChan chan bool, clientMode bool, clientURL, apiKey string) TuiInterface
```

### `runClient` — REST → JSON-RPC

The existing `http.Post(baseURL + "/action?key=start&value=...")` targets a REST
route that no longer exists after the mcp PLAN executes. Replace with
`mcp.Client.Call`:

```go
func runClient(cfg BootstrapConfig) {
    exitChan := make(chan bool)
    mcpPort := "3030"
    if p := os.Getenv("TINYWASM_MCP_PORT"); p != "" {
        mcpPort = p
    }
    baseURL := "http://localhost:" + mcpPort
    clientURL := baseURL + "/logs"

    // Read API key (daemon already created it on its startup)
    apiKey := readAPIKey(cfg.APIKeyPath)

    // TuiFactory now receives apiKey so devtui can attach auth to /logs SSE
    ui := cfg.TuiFactory(exitChan, true, clientURL, apiKey)

    // Tell daemon to start the project — now via JSON-RPC, not REST
    if cfg.StartDir != "" {
        // Dispatch: fire-and-forget, no response needed
        mcp.NewClient(baseURL, apiKey).Dispatch("tinywasm/action", map[string]string{
            "key":   "start",
            "value": cfg.StartDir,
        })
    }

    Start(
        cfg.StartDir,
        cfg.Logger,
        ui,
        cfg.BrowserFactory(ui, exitChan),
        cfg.DB,
        exitChan,
        cfg.ServerFactory,
        cfg.GitHubAuth,
        cfg.GitHandler,
        cfg.GoModHandler,
        false, // headless
        true,  // clientMode
        nil,   // no onProjectReady in client mode
    )
}
```

---

## MODIFY: `app/daemon.go`

### `daemonToolProvider` struct changes

```go
type daemonToolProvider struct {
    cfg           BootstrapConfig
    mcpHandler    *mcp.Handler   // replaces httpSrv *TinywasmHTTP
    ssePub        *SSEPublisher  // new: handles SSE publishing (not HTTP)
    toolProxy     *ProjectToolProxy
    logger        func(messages ...any)
    projectCancel context.CancelFunc
    projectDone   chan struct{}
    projectTui    *HeadlessTUI
    mu            sync.Mutex
    lastPath      string
}
```

### `runDaemon` wiring

```go
// Load or create API key for this daemon instance
apiKey, err := loadOrCreateAPIKey(cfg.APIKeyPath)
if err != nil {
    fmt.Printf("Failed to generate API key: %v\n", err)
    os.Exit(1)
}

// FIX: create proxy BEFORE mcpHandler (root bug — was created after)
proxy := NewProjectToolProxy()

// FIX: proxy is a FIXED provider — MCPServer rebuilds include it
toolProviders := append(cfg.McpToolHandlers, dtp, proxy)
mcpHandler := mcp.NewHandler(mcpConfig, sseHub, toolProviders)
mcpHandler.SetLog(logger)

// Wire auth: ALWAYS explicit — mcp.Handler denies all by default.
// Open mode (no APIKeyPath) is a conscious opt-in, not a silent fallback.
if apiKey != "" {
    mcpHandler.SetAuth(mcp.NewTokenAuthorizer(apiKey))
    mcpHandler.SetAPIKey(apiKey) // written into IDE config Authorization headers
} else {
    mcpHandler.SetAuth(mcp.OpenAuthorizer()) // explicit opt-in: local/trusted environment
}
mcpHandler.ConfigureIDEs()

ssePub := NewSSEPublisher(sseHub)

dtp.mcpHandler = mcpHandler
dtp.ssePub = ssePub
dtp.toolProxy = proxy

// Method names defined here in app — mcp.Handler is agnostic.
const (
    methodAction = "tinywasm/action"
    methodState  = "tinywasm/state"
)

// Register action dispatcher — app owns the method name and param schema.
mcpHandler.RegisterMethod(methodAction, func(ctx context.Context, params []byte) (any, error) {
    var p struct {
        Key   string `json:"key"`
        Value string `json:"value"`
    }
    json.Unmarshal(params, &p)

    dtp.mu.Lock()
    projectTui := dtp.projectTui
    dtp.mu.Unlock()
    if projectTui != nil && projectTui.DispatchAction(p.Key, p.Value) { return okResult, nil }
    if ui.DispatchAction(p.Key, p.Value) { return okResult, nil }
    switch p.Key {
    case "start":
        if p.Value != "" { go dtp.startProject(p.Value) }
    case "stop":
        dtp.stopProject()
    case "restart":
        dtp.restartCurrentProject()
    default:
        logger("Unknown UI action:", p.Key)
    }
    return okResult, nil
})

// Register state provider — app owns the method name and return schema.
mcpHandler.RegisterMethod(methodState, func(ctx context.Context, _ []byte) (any, error) {
    dtp.mu.Lock()
    projectTui := dtp.projectTui
    dtp.mu.Unlock()
    if projectTui != nil { return json.RawMessage(projectTui.GetHandlerStates()), nil }
    return json.RawMessage(ui.GetHandlerStates()), nil
})

mcpHandler.Serve(exitChan)
```

### `runProjectLoop` — RelayLog + onProjectReady + cleanup

```go
headlessTui.RelayLog = func(tabTitle, handlerName, color, msg string) {
    d.ssePub.PublishTabLog(tabTitle, handlerName, color, msg)
}

// onProjectReady: activate proxy then trigger MCPServer rebuild
onProjectReady := func(h *Handler) {
    providers := buildProjectProviders(h)
    d.toolProxy.SetActive(providers...)
    // SetDynamicProviders() triggers rebuildMCPServer() which re-reads all
    // fixed providers (including proxy, now populated with project tools).
    d.mcpHandler.SetDynamicProviders()
    d.logger("ProjectToolProxy activated:", len(providers), "providers")
    d.ssePub.PublishStateRefresh() // signal only — devtui re-fetches via JSON-RPC
}

// defer cleanup: clear proxy then trigger rebuild
defer func() {
    d.toolProxy.SetActive()
    d.mcpHandler.SetDynamicProviders() // rebuild with empty proxy
    d.logger("Project loop cleanup: proxy cleared")
}()
```

---

## MODIFY: `app/start.go` (standalone mode)

```go
// Before (bad refactor):
h.HTTP = NewTinywasmHTTP(mcpPort, mcpHandler.HTTPHandler(), sseHub, "")
h.HTTP.OnState(...)
h.Tui.AddHandler(h.HTTP, ...)
go h.HTTP.Serve(h.ExitChan)

// After (restored):
ssePub := NewSSEPublisher(sseHub)
h.Logger = func(messages ...any) {
    logger(messages...)
    ssePub.PublishLog(fmt.Sprint(messages...))
}

toolHandlers := buildProjectProviders(h)
toolHandlers = append(toolHandlers, mcpToolHandlers...)

h.MCP = mcp.NewHandler(mcpConfig, sseHub, toolHandlers)
h.MCP.SetLog(logger)
h.MCP.ConfigureIDEs()
h.MCP.SetAuth(mcp.OpenAuthorizer()) // standalone: local only, explicit opt-in
// Register state method — same name as daemon so mcp.Client callers work identically
h.MCP.RegisterMethod(methodState, func(_ context.Context, _ []byte) (any, error) {
    return json.RawMessage(h.Tui.GetHandlerStates()), nil
})
// No methodAction in standalone — handlers dispatch locally via TUI

h.Tui.AddHandler(h.MCP, colorOrangeLight, h.SectionBuild)
SetActiveHandler(h)
wg.Add(1)
go func() {
    defer wg.Done()
    h.MCP.Serve(h.ExitChan)
}()
```

---

## Files to Create / Modify / Delete

| File | Action | Description |
|------|--------|-------------|
| `app/http_server.go` | **DELETE** | TinywasmHTTP removed entirely |
| `app/sse_publisher.go` | **CREATE** | `SSEPublisher`, `LogEntry`, `ssePublisher` interface |
| `app/api_key.go` | **CREATE** | `generateAPIKey`, `loadOrCreateAPIKey`, `readAPIKey` |
| `app/handler.go` | **MODIFY** | `HTTP *TinywasmHTTP` → `MCP *mcp.Handler` |
| `app/bootstrap.go` | **MODIFY** | Add `APIKeyPath` to `BootstrapConfig`; update `TuiFactory` signature (add `apiKey`); fix `runClient` to use `mcp.Client` + `readAPIKey` |
| `app/daemon.go` | **MODIFY** | Fix proxy order, replace httpSrv→mcpHandler+ssePub, wire SSEPublisher, wire tokenAuthorizer, fix onProjectReady/cleanup |
| `app/start.go` | **MODIFY** | Restore `h.MCP = mcp.NewHandler(...)`, SSEPublisher for logger |
| `app/mcp_registry.go` | NO CHANGE | ProjectToolProxy unchanged |
| `app/test/mcp_test.go` | **MODIFY** | Update `mcp.NewHandler` to 3-arg: `(config, sseHub, providers)` |

---

## Execution Steps

### Step 1 — Prerequisite: update tinywasm/mcp to v0.0.17
```bash
go get github.com/tinywasm/mcp@v0.0.17
```
Confirm the following exist at v0.0.17:
- `mcp.NewHandler(config Config, sseHub SSEHub, fixedProviders []ToolProvider)`
- `mcp.NewClient(baseURL, apiKey string) *Client`
- `mcp.NewTokenAuthorizer(token string) Authorizer`
- `mcp.OpenAuthorizer() Authorizer`

### Step 2 — Create `app/api_key.go`

### Step 3 — Create `app/sse_publisher.go`

### Step 4 — Delete `app/http_server.go`

### Step 5 — Modify `app/handler.go`

### Step 6 — Modify `app/bootstrap.go`
Update `BootstrapConfig.TuiFactory` signature; add `APIKeyPath`; fix `runClient`.

### Step 7 — Modify `app/daemon.go`

### Step 8 — Modify `app/start.go`

### Step 9 — Update `app/test/mcp_test.go`

### Step 10 — Run tests and publish
```bash
gotest
gopush 'fix: remove TinywasmHTTP, restore mcp.Handler, fix ProjectToolProxy wiring, token auth, JSON-RPC runClient'
```

---

## Test Strategy

| Test | Validates |
|------|-----------|
| `TestDaemon_ProxyWired_ToolsLoadAfterProjectStart` | proxy in fixed providers → project tools appear after onProjectReady |
| `TestDaemon_ProjectStop_ClearsProjectTools` | After cleanup, project tools disappear, daemon tools remain |
| `TestSSEPublisher_PublishTabLog_CorrectJSONFields` | `tab_title`, `handler_name`, `handler_color`, `handler_type` in JSON |
| `TestSSEPublisher_PublishStateRefresh_SendsSignal` | hub receives `Publish` with `{"handler_type":0}` |
| `TestStart_StandaloneMode_MCPHandlerSet` | `h.MCP` not nil in non-headless mode |
| `TestDaemon_NoAPIKeyPath_UsesOpenAuthorizer` | Empty `APIKeyPath` → `SetAuth(OpenAuthorizer())` called explicitly |
| `TestDaemon_WithAPIKeyPath_UsesTokenAuthorizer` | Non-empty `APIKeyPath` → `SetAuth(TokenAuthorizer)` called |
| `TestGenerateAPIKey_Length` | Returns 64-char hex string (32 bytes) |
| `TestLoadOrCreateAPIKey_EmptyPath_ReturnsEmpty` | Empty path → `("", nil)` |
| `TestLoadOrCreateAPIKey_CreatesOnMissing` | Missing file → creates + returns key |
| `TestLoadOrCreateAPIKey_ReadsExisting` | Existing file → returns stored key |
| `TestReadAPIKey_EmptyPath_ReturnsEmpty` | Empty path → `""` |
| `TestRunClient_JSONRPCStartCall` | POST to /mcp with `tinywasm/action` + `key=start` (mocked handler) |
| Existing `test/mcp_test.go` | Must pass with updated 3-arg signature |
