# PLAN: Fix TUI Footer Handlers Not Appearing on Startup

## Problem

When `tinywasm` starts, the TUI footer shows no interactive handlers (no buttons, no editable
fields). The user can only view logs but cannot perform any actions.

## Root Cause

**File:** `start.go` lines 114–126

The external `onProjectReady` callback (passed by the daemon) is called **before**
`h.OnProjectReady(&wg)`, which internally calls `InitBuildHandlers()`.

```
start.go:115  onProjectReady(h)       ← fires FIRST → PublishStateRefresh() with empty HeadlessTUI
start.go:125  h.OnProjectReady(&wg)   ← fires SECOND → InitBuildHandlers() registers all handlers
```

The daemon's `onProjectReady` callback (`daemon.go:507-519`) does two things:

1. `buildProjectProviders(h)` — uses `h.WasmClient`, which is **nil** at this point
2. `d.ssePub.PublishStateRefresh()` — signals the client TUI to re-fetch handler state

When the client receives the state refresh signal and calls `GET /tinywasm/state`, the
`HeadlessTUI.handlers` slice is still empty (handlers have not been registered yet).
The client calls `clearRemoteHandlers()` + `reconstructRemoteHandlers([])` — wiping any
previously cached handlers and adding nothing.

`InitBuildHandlers()` runs afterward and registers all six handlers with `HeadlessTUI`, but no
second `PublishStateRefresh()` is ever emitted. The client TUI never learns about the handlers
and the footer stays permanently empty.

## Affected Files

| File | Lines | Change |
|---|---|---|
| `start.go` | 114–126 | Move `onProjectReady(h)` to **after** `h.OnProjectReady(&wg)` |

## Fix

Move the external callback invocation so it always fires **after** `InitBuildHandlers()` has
completed. This guarantees:

- `h.WasmClient` is initialized before `buildProjectProviders(h)` uses it
- `HeadlessTUI.handlers` is fully populated before `PublishStateRefresh()` fires
- The client fetches non-empty state and reconstructs all footer handlers

### Current code (`start.go` lines 114–126)

```go
// Call onProjectReady callback (used by daemon to set up tool proxy)
if onProjectReady != nil {
    onProjectReady(h)
}

if !h.IsPartOfProject() {
    sectionWizard := h.AddSectionWIZARD(func() {
        h.OnProjectReady(&wg)
    })
    h.Tui.SetActiveTab(sectionWizard)
} else {
    h.OnProjectReady(&wg)
}
```

### Fixed code

```go
if !h.IsPartOfProject() {
    sectionWizard := h.AddSectionWIZARD(func() {
        h.OnProjectReady(&wg)
        // Fire external callback AFTER InitBuildHandlers has run
        if onProjectReady != nil {
            onProjectReady(h)
        }
    })
    h.Tui.SetActiveTab(sectionWizard)
} else {
    h.OnProjectReady(&wg)
    // Fire external callback AFTER InitBuildHandlers has run
    if onProjectReady != nil {
        onProjectReady(h)
    }
}
```

## Why This Works

- `h.OnProjectReady(&wg)` → `InitBuildHandlers()` → registers WasmClient, Server,
  AssetsHandler, Watcher, Config, Browser with `HeadlessTUI` via `AddHandler`
- External callback runs next: `buildProjectProviders(h)` finds `h.WasmClient != nil`
- `PublishStateRefresh()` fires: client calls `GET /tinywasm/state`
- `HeadlessTUI.GetHandlerStates()` returns all six handlers (htEdit for WasmClient/Server,
  htDisplay for the rest)
- Client reconstructs remote fields → `FieldHandlers` populated → footer renders correctly

## Execution Steps

1. Open `start.go`
2. Remove the block at lines 114–117 (`if onProjectReady != nil { onProjectReady(h) }`)
3. In the `else` branch (line 124–126), add the callback call after `h.OnProjectReady(&wg)`
4. In the wizard callback (line 121), add the callback call after `h.OnProjectReady(&wg)`
5. Run existing tests: `go test ./...`
6. Manually verify: start `tinywasm` in a project dir, confirm footer shows handler buttons

## No Other Files Changed in This Library

The `HeadlessTUI` (`headless_tui.go`) and daemon handler (`daemon.go`) are correct as-is.
The daemon sets `d.projectTui = headlessTui` before `Start()` which is correct; the only
problem was the callback ordering inside `Start()`.
