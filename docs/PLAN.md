# PLAN: Fix Client TUI — BUILD & DEPLOY Tabs Not Showing Logs

**Status**: Ready for execution
**Project**: `tinywasm/app`
**Related Docs**: [ARCHITECTURE](ARQUITECTURE.md) | [MCP Architecture Plan](MCP_ARCHITECTURE_PLAN.md) | [MCP Refactor Flow](diagrams/MCP_REFACTOR_FLOW.md)

---

## Root Cause Summary

After the MCP refactor, running `tinywasm` (client mode) shows only tab 0 (SHORTCUTS/help). Tabs 1 (BUILD) and 2 (DEPLOY) are visible but completely empty. Four separate bugs interact to cause this:

---

## Bug Analysis

### BUG-1 (PRIMARY): TuiFactory never enables SSE client in the TUI

**File**: `app/cmd/tinywasm/main.go:73-81`

```go
// CURRENT (broken): no ClientMode or ClientURL passed
TuiFactory: func(exitChan chan bool) app.TuiInterface {
    return devtui.NewTUI(&devtui.TuiConfig{
        AppName:    "TINYWASM",
        ExitChan:   exitChan,
        // ClientMode: false (default)
        // ClientURL:  "" (default)
    })
},
```

In `devtui/init.go:127-129`:
```go
if c.ClientMode && c.ClientURL != "" {
    go tui.startSSEClient(c.ClientURL)  // NEVER STARTS
}
```

**Effect**: The SSE client goroutine is never launched. Client TUI never connects to `http://localhost:3030/logs`. Tabs 1 and 2 receive no data.

---

### BUG-2 (SECONDARY): SSE data format mismatch

**Files**:
- `mcpserve/handler.go:238-242` — publishes raw strings: `h.sseHub.Publish([]byte(msg), "logs")`
- `devtui/sse_client.go:108-114` — expects JSON `tabContentDTO`:

```go
var dto tabContentDTO
if err := json.Unmarshal([]byte(data), &dto); err != nil {
    continue  // silently drops all raw string messages
}
// section lookup by tab_title:
if section == nil {
    continue  // drops message if tab_title doesn't match a known section
}
```

**Effect**: Even if the SSE connection was established (fixing BUG-1), all messages would be silently dropped due to JSON parse failure.

---

### BUG-3 (STRUCTURAL): Project component logs never reach daemon SSE hub

**Files**: `app/bootstrap.go:283-331`, `app/start.go:107-114`, `app/start.go:190-225`

When `start_development` is called:

1. `daemonToolProvider.runProjectLoop` calls `app.Start` with `d.logger` (stdout/file logger).
2. `app.Start` wraps `h.Logger` to also call `h.MCP.PublishLog()`.
3. BUT `app.Start` also creates its own MCP (`h.MCP`) on port **3030** — already occupied by the daemon MCP.
4. The project MCP's `sseHub` is initialized but no client connects to it (clients connect to the daemon MCP on 3030).
5. The daemon's SSE hub (`mcpHandler.sseHub`) never receives any project component logs.

**Effect**: The daemon's SSE stream is empty of project logs. Even with BUG-1 and BUG-2 fixed, the client would see nothing useful in BUILD/DEPLOY.

---

### BUG-4 (STRUCTURAL): HeadlessTUI discards AddHandler — no logger injected into components

**File**: `app/headless_tui.go:25-28`

```go
func (t *HeadlessTUI) AddHandler(Handler any, color string, tabSection any) {
    // Does nothing — no logger is injected into components
}
```

In daemon mode, `app.Start` calls `h.Tui.AddHandler(h.WasmClient, ...)`. With `HeadlessTUI`, this call is a no-op. The devtui logger injection (`handler.SetLog(...)`) never happens. Components like `WasmClient`, `Server`, `Watcher`, etc., have no logger attached and produce no output.

**Effect**: Even if BUG-3 is fixed, the relay has nothing to relay because components never log.

---

## Fix Plan

### Step 1 — Fix BUG-1: Add `clientMode` to TuiFactory signature

**Files**: `app/bootstrap.go`, `app/cmd/tinywasm/main.go`

**1.1** Change `TuiFactory` type in `BootstrapConfig` (`bootstrap.go`):

```go
TuiFactory func(exitChan chan bool, clientMode bool, clientURL string) TuiInterface
```

**1.2** Update `runClient()` to pass `clientMode=true` and the SSE URL:

```go
func runClient(cfg BootstrapConfig) {
    exitChan := make(chan bool)
    mcpPort := "3030"
    if p := os.Getenv("TINYWASM_MCP_PORT"); p != "" {
        mcpPort = p
    }
    clientURL := "http://localhost:" + mcpPort + "/logs"
    ui := cfg.TuiFactory(exitChan, true, clientURL)
    // rest unchanged...
}
```

**1.3** Update `runDaemon()` to pass `clientMode=false`:

```go
if cfg.TuiFactory != nil {
    ui = cfg.TuiFactory(exitChan, false, "")
} else {
    ui = NewHeadlessTUI(logger)
}
```

**1.4** Update `main.go` TuiFactory implementation:

```go
TuiFactory: func(exitChan chan bool, clientMode bool, clientURL string) app.TuiInterface {
    return devtui.NewTUI(&devtui.TuiConfig{
        AppName:    "TINYWASM",
        AppVersion: Version,
        ExitChan:   exitChan,
        Color:      devtui.DefaultPalette(),
        Logger:     func(messages ...any) { logger.Logger(messages...) },
        Debug:      *debugFlag,
        ClientMode: clientMode,
        ClientURL:  clientURL,
    })
},
```

---

### Step 2 — Fix BUG-2: Align SSE data format — publisher must send JSON DTOs

**Files**: `mcpserve/handler.go`

**2.1** Add a `LogEntry` struct to `mcpserve/handler.go` matching the `tabContentDTO` JSON keys that `devtui/sse_client.go` expects:

```go
// LogEntry matches devtui.tabContentDTO JSON structure for SSE routing.
type LogEntry struct {
    Id           string `json:"id"`
    Timestamp    string `json:"timestamp"`
    Content      string `json:"content"`
    Type         string `json:"type"`
    TabTitle     string `json:"tab_title"`
    HandlerName  string `json:"handler_name"`
    HandlerColor string `json:"handler_color"`
    HandlerType  int    `json:"handler_type"` // 0 = loggable
}
```

**2.2** Add `PublishTabLog(tabTitle, handlerName, handlerColor, msg string)`:

```go
func (h *Handler) PublishTabLog(tabTitle, handlerName, handlerColor, msg string) {
    if h.sseHub == nil {
        return
    }
    entry := LogEntry{
        Id:           fmt.Sprintf("%d", time.Now().UnixNano()),
        Timestamp:    time.Now().Format("15:04:05"),
        Content:      msg,
        Type:         "info",
        TabTitle:     tabTitle,
        HandlerName:  handlerName,
        HandlerColor: handlerColor,
        HandlerType:  0,
    }
    data, err := json.Marshal(entry)
    if err != nil {
        return
    }
    h.sseHub.Publish(data, "logs")
}
```

**2.3** Update `PublishLog` to use `PublishTabLog` with a default BUILD routing:

```go
func (h *Handler) PublishLog(msg string) {
    h.PublishTabLog("BUILD", "MCP", "#f97316", msg) // orange for MCP
}
```

---

### Step 3 — Fix BUG-3 and BUG-4: HeadlessTUI relays structured logs to daemon SSE

**Files**: `app/headless_tui.go`, `app/bootstrap.go`, `app/start.go`

**3.1** Add `RelayLog` field and fix `NewTabSection` in `headless_tui.go`:

```go
type headlessSection struct {
    Title string
}

func (s *headlessSection) GetTitle() string { return s.Title }

type HeadlessTUI struct {
    logger   func(messages ...any)
    RelayLog func(tabTitle, handlerName, color, msg string) // optional SSE relay
}

func NewHeadlessTUI(logger func(messages ...any)) *HeadlessTUI {
    return &HeadlessTUI{logger: logger}
}

func (t *HeadlessTUI) NewTabSection(title, description string) any {
    return &headlessSection{Title: title}
}
```

**3.2** Implement smart `AddHandler` that injects loggers into components:

```go
func (t *HeadlessTUI) AddHandler(handler any, color string, tabSection any) {
    // Extract tab title
    tabTitle := "BUILD"
    type TitleGetter interface{ GetTitle() string }
    if ts, ok := tabSection.(TitleGetter); ok {
        tabTitle = ts.GetTitle()
    }

    // Extract handler name
    type Namer interface{ Name() string }
    handlerName := "HANDLER"
    if n, ok := handler.(Namer); ok {
        handlerName = n.Name()
    }

    // Inject logger
    type LogSetter interface{ SetLog(func(...any)) }
    if s, ok := handler.(LogSetter); ok {
        s.SetLog(func(messages ...any) {
            msg := fmt.Sprint(messages...)
            if t.logger != nil {
                t.logger(msg)
            }
            if t.RelayLog != nil {
                t.RelayLog(tabTitle, handlerName, color, msg)
            }
        })
    }
}
```

**3.3** In `bootstrap.go — runProjectLoop`, wire `RelayLog` to daemon MCP:

```go
func (d *daemonToolProvider) runProjectLoop(ctx context.Context, projectPath string) {
    runExitChan := make(chan bool)

    headlessTui := NewHeadlessTUI(d.logger)
    headlessTui.RelayLog = func(tabTitle, handlerName, color, msg string) {
        d.mcpHandler.PublishTabLog(tabTitle, handlerName, color, msg)
    }

    browser := d.cfg.BrowserFactory(headlessTui, runExitChan)
    // ... goroutine and Start() call unchanged
}
```

**3.4** Skip project-level MCP creation in `start.go` when `headless=true` (avoids port conflict):

Wrap the MCP creation block (lines ~190-225) with `if !headless { ... }`. Adjust `wg.Add()` accordingly so the WaitGroup count stays correct:

```go
// start.go — existing wg.Add(2) at line 132 covers UI + MCP
// Change to only add MCP counter when not headless:
wg.Add(1) // UI always
if !headless {
    wg.Add(1) // MCP server (only in standalone mode)
    // ... full MCP setup and Serve() goroutine
} // else: no project MCP, no port conflict
```

---

### Step 4 — Verification

**4.1** Manual flow test:
1. Start daemon: `tinywasm -mcp`
2. Call `start_development` via MCP tool for any local tinywasm project.
3. Open client: `tinywasm` (no flags).
4. Confirm tabs 1 (BUILD) and 2 (DEPLOY) populate with color-coded logs.

**4.2** Run test suite: `gotest`

**4.3** Verify no regression in standalone mode (without daemon): `tinywasm -mcp` from project dir should still work as before.

---

## Files to Modify (Summary)

| File | Change |
|------|--------|
| `app/cmd/tinywasm/main.go` | Update `TuiFactory` signature; add `ClientMode`/`ClientURL` to devtui config |
| `app/bootstrap.go` | Change `TuiFactory` type; update `runClient()`, `runDaemon()`, `runProjectLoop()` |
| `app/headless_tui.go` | Add `RelayLog`; add `headlessSection`; implement smart `AddHandler` with logger injection |
| `app/start.go` | Skip project-level MCP creation when `headless=true`; fix `wg.Add` count |
| `mcpserve/handler.go` | Add `LogEntry` struct; add `PublishTabLog()`; update `PublishLog()` to use it |

---

## Constraints

- No new external packages.
- No WASM-side changes needed.
- `TuiFactory` signature change is confined to `main.go` (single injection point, per DI rules).
- `mcpserve.LogEntry` JSON keys must exactly match `devtui.tabContentDTO` JSON tags.
