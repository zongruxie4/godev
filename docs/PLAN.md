# PLAN: tinywasm/app — Migration to New tinywasm/mcp API

> Date: 2026-03-28 (revisado 2026-04-08)
> Blocked by: `tinywasm/mcp` Stages 2–4
> Execute when: `tinywasm/mcp` Stage 4 is complete

---

## Context

`tinywasm/mcp` was refactored. The API changed completely. `tinywasm/app` compiles against the old version and is broken. This plan migrates `tinywasm/app` to the new API in the correct order to minimize time spent with broken code.

Crucially, **IDE auto-configuration (ConfigureIDEs)** has been architecturally moved out of `tinywasm/mcp` (which must remain pure protocol) and into `tinywasm/app` (which orchestrates the local developer environment).

---

## New mcp API — Confirmed Signatures (v0.1.1)

> Verified against `/home/cesar/go/pkg/mod/github.com/tinywasm/mcp@v0.1.1/`

```go
// Config — Port/ServerName/AppName removed; Auth required at construction time
type Config struct {
    Name    string
    Version string
    Auth    Authorizer   // required — nil returns error
    SSE     SSEPublisher // optional
}

// SSEPublisher — single-channel signature (NOT variadic)
type SSEPublisher interface {
    Publish(data []byte, channel string)
}
// NOTE: *sse.SSEServer satisfies this directly — no adapter needed.

// Tool — new structure
type Tool struct {
    Name        string
    Description string
    InputSchema string // JSON schema string
    Resource    string // required — RBAC resource
    Action      byte   // required — 'c','r','u','d'
    Execute     func(ctx *context.Context, req Request) (*Result, error)
}

// ToolProvider — renamed method
type ToolProvider interface {
    Tools() []Tool  // was GetMCPTools()
}

// Server — only public methods:
//   NewServer(config Config, providers []ToolProvider) (*Server, error)
//   (*Server).AddTool(tool Tool) error
//   (*Server).HandleMessage(ctx *context.Context, message []byte) JSONRPCMessage

// Client — uses *tinywasm/context.Context (NOT stdlib context.Context)
func NewClient(baseURL, apiKey string) *Client
func (c *Client) Dispatch(ctx *context.Context, method string, params any)
func (c *Client) Call(ctx *context.Context, method string, params any, callback func([]byte, error))
```

**Methods that NO LONGER EXIST** (deleted from mcp.Handler/mcp.Server):
- `mcp.NewHandler(...)` → replaced by `mcp.NewServer(...)`
- `(*Server).RegisterMethod(name, fn)` → **gone**; app must use native `net/http` handlers
- `(*Server).SetDynamicProviders(providers...)` → **gone**; app must re-create or call `AddTool`
- `(*Server).SetAuth(...)` → **gone**; Auth is set at `mcp.Config` construction time
- `(*Server).SetAPIKey(...)` → **gone**; moved to app layer
- `(*Server).SetLog(...)` → **gone**
- `(*Server).Serve(exitChan)` → **gone**; app owns the HTTP server lifecycle
- `(*Server).ConfigureIDEs()` → **gone**; moved to `app/mcp_ide`
- `(*Server).RegisterRoutes(mux)` → **does NOT exist**; app exposes `HandleMessage` manually
- `mcp.Parameter`, `mcp.Handler` types → **gone**

---

## Complete Inventory of Incompatibilities

| File | Old Usage | Replacement |
|---------|-----------|-----------|
| `daemon.go:30-35` | `mcp.Config{Port, ServerName, ServerVersion, AppName, AppVersion}` | `mcp.Config{Name, Version, Auth, SSE}` |
| `daemon.go:71` | `mcp.NewHandler(config, sseHub, providers)` | `mcp.NewServer(config, providers)` returns `(*Server, error)` |
| `daemon.go:77` | `mcpHandler.SetAuth(mcp.NewTokenAuthorizer(apiKey))` | `mcp.Config{Auth: mcp.NewTokenAuthorizer(apiKey)}` |
| `daemon.go:78` | `mcpHandler.SetAPIKey(apiKey)` | Handled by `mcp_ide` package |
| `daemon.go:80` | `mcpHandler.SetAuth(mcp.OpenAuthorizer())` | `mcp.Config{Auth: mcp.OpenAuthorizer()}` |
| `daemon.go:82` | `mcpHandler.ConfigureIDEs()` | `mcp_ide.Configure(appName, port, apiKey)` in boot |
| `daemon.go:72` | `mcpHandler.SetLog(logger)` | Removed — no equivalent in new API |
| `daemon.go:98,133` | `mcpHandler.RegisterMethod(name, fn)` | Native `http.HandleFunc` on app mux (see Stage 3) |
| `daemon.go:144` | `mcpHandler.Serve(exitChan)` | App owns HTTP server — wrap `mcpServer.HandleMessage` in a handler |
| `daemon.go:151` | `*mcp.Handler` field type | `*mcp.Server` |
| `daemon.go:169` | `mcp.Tool{Parameters []mcp.Parameter, Execute func(map[string]any)}` | `mcp.Tool{InputSchema string, Resource, Action, Execute func(*ctx, Request)(*Result,error)}` |
| `daemon.go:313,325` | `d.mcpHandler.SetDynamicProviders(...)` | **No equivalent** — must call `mcpServer.AddTool` for each new tool, or store `*mcp.Server` ref and rebuild |
| `sse_adapter.go` | `sseHubAdapter` wrapping `*sse.SSEServer` | Remove — `*sse.SSEServer` satisfies `mcp.SSEPublisher` directly |
| `handler.go:50` | `MCP *mcp.Handler` | `MCP *mcp.Server` |
| `mcp-tools.go:9` | `mcp.Tool{Parameters []mcp.Parameter, Execute func(...)}` | New Tool struct |
| `mcp_registry.go:39` | `GetMCPTools() []mcp.Tool` | `Tools() []mcp.Tool` |
| `interface.go:43` | `GetMCPTools() []mcp.Tool` | `Tools() []mcp.Tool` |
| `bootstrap.go:88` | `context.Background()` (stdlib) in `mcp.NewClient().Dispatch` | `*tinywasm/context.Context` — import `github.com/tinywasm/context` and pass `ctx.Background()` or equivalent |

---

## Critical Issues NOT Previously Documented

### 1. `bootstrap.go:88` — Context type mismatch (NEW error)
`mcp.Client.Dispatch` requires `*github.com/tinywasm/context.Context`, not stdlib `context.Context`.
```go
// Current (broken)
mcp.NewClient(baseURL, apiKey).Dispatch(context.Background(), ...)

// Fix
import twctx "github.com/tinywasm/context"
mcp.NewClient(baseURL, apiKey).Dispatch(twctx.Background(), ...)
```

### 2. `daemon.go:313,325` — `SetDynamicProviders` gone (NO REPLACEMENT IN API)
`daemonToolProvider` currently calls `d.mcpHandler.SetDynamicProviders(providers...)` when a project starts/stops. The new `mcp.Server` has no such method. Possible solutions:
- **Option A** (recommended): Store `*mcp.Server` and call `AddTool` for each new tool after project switch. Clear by rebuilding the server.
- **Option B**: Rebuild `*mcp.Server` entirely on each project switch (heavier but simpler).
The plan **must resolve this before Stage 4**.

### 3. `daemon.go:144` — `Serve(exitChan)` gone
The app must implement its own HTTP server wrapping `mcpServer.HandleMessage`. The existing `tinySSE` mux or a new `net/http.ServeMux` must:
- `POST /mcp` → extract auth token from `Authorization` header, call `mcpServer.HandleMessage`
- `GET /logs` → SSE endpoint (already handled by `tinySSE`)

### 4. `mcp.EmptyArgs` does NOT exist in v0.1.1
Stage 4 references `new(mcp.EmptyArgs).Schema()` — this type is absent from the package. InputSchema must be a raw JSON schema string or generated via `ormc gen`. Verify this before executing Stage 4.

---

## Implementation Stages

For an orderly execution and review, the plan has been divided into the following modules:

### [Stage 1 — Preparation & Dependencies](stages/stage1_preparation.md)
Verify MCP API readiness and update `go.mod`. ✓ `mcp v0.1.1` already in `go.mod`.

### [Stage 2 — Remove sseHubAdapter](stages/stage2_remove_sse_adapter.md)
Simplify SSE injection via `mcp.SSEPublisher`.
> **Correction**: Stage 2 refers to `mcp.SSETransport` — the actual interface is `mcp.SSEPublisher`.

### [Stage 3 — Migrate Config, Routing, and HTTP Server](stages/stage3_config_routing.md)
Update `mcp.Config` usage, implement HTTP handler for `/mcp` using `mcpServer.HandleMessage`, replace `RegisterMethod` with native `http.HandleFunc`, and replace `Serve(exitChan)` with app-owned `http.ListenAndServe`.
> **Gap in stage**: `RegisterRoutes` was referenced but does NOT exist. App must wire `HandleMessage` manually.
> **Gap in stage**: `SetDynamicProviders` must be resolved here (see Critical Issue #2).

### [Stage 4 — Migrate Tool Definitions & Provider Protocol](stages/stage4_tool_definitions.md)
Update `mcp.Tool` struct definitions, define schemas, and migrate `GetMCPTools()` to `Tools()`.
> **Verify**: `mcp.EmptyArgs` may not exist — use raw JSON schema strings instead.

### [Stage 5 — IDE Configuration Migration](stages/stage5_ide_configuration.md)
Extract and implement `ConfigureIDEs` within `tinywasm/app/mcp_ide` instead of `mcp`, recovering tests and config payloads.

### [Stage 6 — Integration Tests](stages/stage6_tests.md)
Restore and update full testing suite for daemon bootstrapping, Tool calls, and native HTTP methods.

---

## Execution Order

```
Stage 1 (verify prereqs — mcp v0.1.1 already present)
    ↓
Stage 2 (remove sseHubAdapter) + Stage 3 (config, routing, HTTP server, SetDynamicProviders) — in parallel
    ↓
Stage 4 (tool definitions — verify EmptyArgs first)
    ↓
Stage 5 (IDE Configuration extraction)
    ↓
Stage 6 (Integration Tests & HTTP route validation)
```
