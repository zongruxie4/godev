# PLAN Stage 4: Clean Shutdown — tinywasm/app

## Prerequisite

**Stage 2 must be complete before this stage.**
Stage 2 adds log-to-file. Without it, goroutines that log during shutdown (server, watcher,
browser) write to stdout after the terminal is restored — partial fix only.

**devtui PLAN_clean_shutdown.md must be applied and published before this stage.**
This stage adapts the app side to the new devtui contract (no ExitChan in TuiConfig,
Shutdown() method available on TuiInterface).

## Problem

After devtui's shutdown fix, the terminal restore is correct on the devtui side. But
tinywasm/app still has issues in its shutdown flow:

1. `TuiFactory` passes `exitChan` to `devtui.NewTUI` via `TuiConfig.ExitChan` — field removed
   in devtui; must be removed from the factory call.
2. `runClient` in `bootstrap.go` closes `exitChan` BEFORE `Start()` returns — inverted: goroutines
   (server, watcher, browser) stop before the TUI cleanup completes, causing log writes during
   the restore window.
3. The daemon "quit" POST uses `http.DefaultClient` (infinite timeout) — if the daemon is
   slow or dead, `runClient` blocks indefinitely after the terminal is restored.
4. `cmd/tinywasm/main.go` must wire OS signal handling (SIGINT/SIGTERM) via `tui.Shutdown()`
   so VSCode terminal close and `kill` commands trigger clean exit (not just Ctrl+C).

## Design

The correct shutdown ownership chain:

```
devtui (owns terminal restore)
  → Start() returns only after terminal is clean and SSE goroutine is done
    → runClient closes exitChan
      → server, watcher, browser goroutines stop (write to FILE log, not stdout)
        → POST "quit" to daemon (500ms timeout, best-effort)
          → process exits
```

OS signals are handled in `main.go` via a goroutine that calls `tui.Shutdown()`.

## Affected Files

| File | Change |
|------|--------|
| `cmd/tinywasm/main.go` | Wire SIGINT/SIGTERM → `tui.Shutdown()` |
| `bootstrap.go` | Remove `ExitChan` from `TuiFactory` call; close `exitChan` AFTER `Start()` returns; add 500ms timeout to daemon quit POST |
| `interface.go` | Add `Shutdown()` to `TuiInterface` |
| `headless_tui.go` | Implement `Shutdown()` as no-op |
| `start.go` | No change needed — `Start()` already blocks on `clientWg.Wait()` |

## Implementation Steps

### Step 1 — `interface.go`: add Shutdown() to TuiInterface

```go
type TuiInterface interface {
    NewTabSection(title, description string) any
    AddHandler(Handler any, color string, tabSection any)
    Start(syncWaitGroup ...any)
    RefreshUI()
    ReturnFocus() error
    SetActiveTab(section any)
    GetHandlerStates() []byte
    DispatchAction(key, value string) bool
    Shutdown() // signals graceful stop; no-op on HeadlessTUI
}
```

### Step 2 — `headless_tui.go`: implement Shutdown() as no-op

```go
func (h *HeadlessTUI) Shutdown() {}
```

### Step 3 — `bootstrap.go`: fix runClient shutdown flow

Current (broken):
```go
func runClient(cfg BootstrapConfig) {
    exitChan := make(chan bool)
    // ...
    ui := cfg.TuiFactory(exitChan, true, clientURL, apiKey)  // ← passes exitChan to devtui
    // ...
    Start(..., exitChan, ..., true)  // Start() launches tui.Start() internally
    // ← exitChan already closed by devtui before we get here
    POST "quit" to daemon  // ← http.DefaultClient, infinite timeout
}
```

Fixed:
```go
func runClient(cfg BootstrapConfig) {
    exitChan := make(chan bool)

    // TuiFactory no longer receives exitChan (removed from devtui.TuiConfig)
    ui := cfg.TuiFactory(true, clientURL, apiKey)

    // ... project start POST to daemon ...

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
        false,
        true,
        nil,
    )
    // Start() returns here only after devtui has fully restored the terminal.
    // NOW it is safe to close exitChan — no goroutine will write to stdout after this.
    close(exitChan)

    // Notify daemon to stop — best-effort, 500ms timeout
    quitBody, _ := json.Marshal(map[string]string{"key": "quit"})
    req, err := http.NewRequest("POST", baseURL+"/tinywasm/action", bytes.NewReader(quitBody))
    if err == nil {
        req.Header.Set("Content-Type", "application/json")
        if apiKey != "" {
            req.Header.Set("Authorization", "Bearer "+apiKey)
        }
        httpClient := &http.Client{Timeout: 500 * time.Millisecond}
        resp, err := httpClient.Do(req)
        if err == nil {
            resp.Body.Close()
        }
    }
}
```

### Step 4 — `bootstrap.go`: OS signal handling in runClient

`ui` is created inside `runClient()`, not in `main.go` — signal handling must live here too.
Add immediately after `ui` is created and before `Start()` is called:

```go
import (
    "os/signal"
    "syscall"
)

func runClient(cfg BootstrapConfig) {
    exitChan := make(chan bool)
    ui := cfg.TuiFactory(true, clientURL, apiKey)

    // Wire OS signals so VSCode terminal close and `kill` trigger clean exit.
    // Ctrl+C is already handled by bubbletea inside devtui; this covers SIGTERM.
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        <-sigCh
        ui.Shutdown()
    }()

    // ... rest of runClient unchanged until Start() call ...
```

Note: the signal goroutine is leaked if the TUI exits via Ctrl+C first. This is safe — the
process is about to exit. The goroutine will be collected by the OS with the process.

### Step 5 — `bootstrap.go` + `daemon.go`: update TuiFactory signature

Remove `exitChan chan bool` as the first parameter from the factory type and all call sites.
The daemon's own `exitChan` (used for its HTTP server lifecycle) is independent and unaffected.

**`bootstrap.go` — BootstrapConfig type:**
```go
// Before:
TuiFactory func(exitChan chan bool, clientMode bool, clientURL, apiKey string) TuiInterface
// After:
TuiFactory func(clientMode bool, clientURL, apiKey string) TuiInterface
```

**`bootstrap.go` — runClient call site:**
```go
// Before:
ui := cfg.TuiFactory(exitChan, true, clientURL, apiKey)
// After:
ui := cfg.TuiFactory(true, clientURL, apiKey)
```

**`daemon.go` — runDaemon call site:**
```go
// Before:
ui = cfg.TuiFactory(exitChan, false, "", "")
// After:
ui = cfg.TuiFactory(false, "", "")
// Note: daemon.go's exitChan continues to exist — it controls the daemon HTTP server
// lifecycle at lines 201, 267, 305. Only the TuiFactory call site changes.
```

**`cmd/tinywasm/main.go` — TuiFactory lambda:**
```go
// Before:
TuiFactory: func(exitChan chan bool, clientMode bool, clientURL, apiKey string) app.TuiInterface {
    return devtui.NewTUI(&devtui.TuiConfig{
        AppName:    "TINYWASM",
        AppVersion: Version,
        ExitChan:   exitChan,   // ← remove this line
        ...
    })
},
// After:
TuiFactory: func(clientMode bool, clientURL, apiKey string) app.TuiInterface {
    return devtui.NewTUI(&devtui.TuiConfig{
        AppName:    "TINYWASM",
        AppVersion: Version,
        // ExitChan removed from devtui.TuiConfig
        Color:      devtui.DefaultPalette(),
        Logger:     func(messages ...any) { logger.Logger(messages...) },
        Debug:      *debugFlag,
        ClientMode: clientMode,
        ClientURL:  clientURL,
        APIKey:     apiKey,
    })
},
```

## Verification

```bash
go build ./...      # must compile without errors
go test ./...       # existing tests must pass
```

Manual scenarios:
1. `tinywasm` → Ctrl+C → terminal must be clean, prompt returns immediately
2. `tinywasm` → close VSCode terminal tab → no zombie process on port 3030
3. `tinywasm` → `kill <pid>` from another terminal → clean exit, no port leak
4. Daemon slow to respond → quit POST times out in 500ms, process still exits cleanly
