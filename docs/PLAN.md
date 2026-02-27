# Plan: Extend TuiInterface for State & Action Dispatch

## Why Extend `TuiInterface`, Not Add a Separate Interface

`TuiInterface` is already the contract between `bootstrap.go` and both TUI
implementations. `bootstrap.go` holds a `ui TuiInterface` reference and calls
`ui.AddHandler(...)` for every registered handler. The TUI already owns the handler
knowledge the moment `AddHandler` is called.

Adding a separate `StateProvider` interface (checked via type assertion) is a design
smell: it means the interface contract is incomplete. The correct answer is to make
the contract complete by extending `TuiInterface` directly. The compiler then
enforces that both `HeadlessTUI` and `DevTUI` implement the full contract — no
runtime assertions, no optional duck-typing.

## Changes to `interface.go`

```go
type TuiInterface interface {
    NewTabSection(title, description string) any
    AddHandler(Handler any, color string, tabSection any)
    Start(syncWaitGroup ...any)
    RefreshUI()
    ReturnFocus() error
    SetActiveTab(section any)

    // GetHandlerStates returns the current handler state as JSON bytes.
    // Format: []StateEntry (devtui wire format).
    // HeadlessTUI: populated by AddHandler calls (daemon mode).
    // DevTUI: returns nil (client mode, not a state server).
    GetHandlerStates() []byte

    // DispatchAction routes a remote action to the handler registered for that key.
    // Returns true if a registered handler handled it; false means caller must handle.
    // HeadlessTUI: iterates handlers slice built in AddHandler, matches by shortcut key.
    // DevTUI: always returns false (actions are sent to daemon, not dispatched locally).
    DispatchAction(key, value string) bool
}
```

## `HeadlessTUI` Additions

`AddHandler` already uses local duck-typed interfaces (`titleGetter`, `namer`,
`logSetter`). The same pattern extends naturally to state capture and action binding.
No new public types. No separate struct hierarchy.

### Internal state (minimal)

```go
// capturedHandler holds everything HeadlessTUI knows about one registered handler.
// The handler reference is retained so GetHandlerStates() reads Value()/Label()
// dynamically — never stale, always reflects current handler state.
type capturedHandler struct {
    tabTitle     string
    handlerName  string
    handlerColor string
    handlerType  int          // htDisplay/htEdit/htExecution/htInteractive
    key          string       // shortcut key (empty if none)
    action       func(string) // called on DispatchAction
    handler      any          // original reference for dynamic Value()/Label() reads
}

type HeadlessTUI struct {
    logger   func(messages ...any)
    RelayLog func(tabTitle, handlerName, color, msg string)
    handlers []capturedHandler // populated by AddHandler
    mu       sync.RWMutex
}
```

> **Why store `handler any` instead of pre-marshaling to `[]byte`?**
> Pre-marshaling at `AddHandler` time captures the initial value only. If the handler's
> value changes later (config edit, browser toggle), `GetHandlerStates()` would return
> stale data. Storing the reference and reading via duck-typing at call time ensures
> the snapshot always reflects current state.

### `AddHandler` extension

After existing log injection, detect interactive capability via local anonymous
interfaces (same duck-typing pattern already used in `AddHandler`) and append to
`handlers`:

```go
// handlerType constants — must match devtui.HandlerType* iota values.
// If devtui/anyHandler.go iota changes, update these in lockstep.
const (
    htDisplay     = 0
    htEdit        = 1
    htExecution   = 2
    htInteractive = 3
)
```

For each handler type detected, capture the action closure and store `capturedHandler`
with the handler reference. The shortcut key drives `DispatchAction` routing.

### `GetHandlerStates() []byte`

Reads current state dynamically from each handler reference. JSON tags match
`devtui.StateEntry` exactly — this is the published wire contract.

```go
func (t *HeadlessTUI) GetHandlerStates() []byte {
    t.mu.RLock()
    defer t.mu.RUnlock()

    // Local duck-typed interfaces — no devtui import required.
    type labeler interface{ Label() string }
    type valuer  interface{ Value() string }

    type stateEntry struct {
        TabTitle     string `json:"tab_title"`
        HandlerName  string `json:"handler_name"`
        HandlerColor string `json:"handler_color"`
        HandlerType  int    `json:"handler_type"`
        Label        string `json:"label"`
        Value        string `json:"value"`
        Shortcut     string `json:"shortcut"`
    }

    entries := make([]stateEntry, 0, len(t.handlers))
    for _, h := range t.handlers {
        e := stateEntry{
            TabTitle:     h.tabTitle,
            HandlerName:  h.handlerName,
            HandlerColor: h.handlerColor,
            HandlerType:  h.handlerType,
            Shortcut:     h.key,
        }
        if l, ok := h.handler.(labeler); ok { e.Label = l.Label() }
        if v, ok := h.handler.(valuer);  ok { e.Value = v.Value() }
        entries = append(entries, e)
    }
    data, _ := json.Marshal(entries)
    return data
}
```

### `DispatchAction(key, value string) bool`

```go
func (t *HeadlessTUI) DispatchAction(key, value string) bool {
    t.mu.RLock()
    defer t.mu.RUnlock()
    for _, h := range t.handlers {
        if h.key == key && h.action != nil {
            go h.action(value) // non-blocking
            return true
        }
    }
    return false
}
```

## `DevTUI` Stubs (devtui package)

Two methods on `*DevTUI`:

```go
func (d *DevTUI) GetHandlerStates() []byte        { return nil }
func (d *DevTUI) DispatchAction(_, _ string) bool { return false }
```

`DevTUI` is a client-mode TUI. It receives state from the daemon (via `GET /state`),
it does not serve it. It forwards actions to the daemon (via `POST /action`), it does
not dispatch them locally.

## Bootstrap Wiring (`bootstrap.go`)

In `runProjectLoop`, `headlessTui` is the local `*HeadlessTUI` variable (which
satisfies `TuiInterface`). The wiring uses the interface — no internal fields accessed.

```go
// Wire state provider — direct interface call, no type assertion
d.mcpHandler.RegisterStateProvider(func() []byte {
    return headlessTui.GetHandlerStates()
})

// Wire action dispatcher — TuiInterface handles the routing
d.mcpHandler.OnUIAction(func(key, value string) {
    if headlessTui.DispatchAction(key, value) {
        return
    }
    switch key {
    case "q": d.logger("Stop command received from UI"); dtp.stopProject()
    case "r": d.logger("Restart command received from UI"); dtp.restartCurrentProject()
    default:  d.logger("Unknown UI action:", key)
    }
})
```

Bootstrap has zero knowledge of `capturedHandler` or any internal `HeadlessTUI` field.
It only uses the `TuiInterface` contract.

## Files to Change

| File | Change |
|------|--------|
| `interface.go` | Add `GetHandlerStates() []byte` and `DispatchAction(key, value string) bool` to `TuiInterface` |
| `headless_tui.go` | Add `capturedHandler` struct; add `handlers []capturedHandler` + `mu`; extend `AddHandler`; implement `GetHandlerStates` and `DispatchAction` |
| `bootstrap.go` | Wire `RegisterStateProvider` and extended `OnUIAction` in `runProjectLoop` |
| `devtui` (separate repo) | Add two stub methods to `DevTUI` |

## Invariants

- `GetHandlerStates()` JSON tags (`tab_title`, `handler_name`, etc.) must match
  `devtui.StateEntry` exactly — documented in `headless_tui.go` as an explicit comment
- `GetHandlerStates()` reads current handler values dynamically — never returns stale
  state regardless of when it is called
- `HeadlessTUI` does NOT import `devtui` (no circular dependency)
- `htDisplay/htEdit/htExecution/htInteractive` constants are unexported, documented
  to mirror `devtui.HandlerType*` iota
- `TuiInterface` adding these methods is backward-compatible only if all existing
  mock implementations (tests) also gain the two stubs

## Test Strategy

- `TestHeadlessTUI_GetHandlerStates_ValidJSON` — JSON output matches devtui.StateEntry
- `TestHeadlessTUI_DispatchAction_ReturnsTrue_WhenHandled`
- `TestHeadlessTUI_DispatchAction_ReturnsFalse_WhenUnknownKey`
- `TestDevTUI_GetHandlerStates_ReturnsNil`
- `TestDevTUI_DispatchAction_ReturnsFalse`
- Update mock `TuiInterface` in test files with the two new stubs

## References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- `interface.go` — `TuiInterface` definition
- Wire format: `devtui/docs/PLAN.md` (`StateEntry`)
- Transport: `mcpserve/docs/PLAN.md`
