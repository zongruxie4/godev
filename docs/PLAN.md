# PLAN: Fix Remote TUI Action Dispatch

## Development Rules

- **DI**: No global state. Interfaces for dependencies. Injection only in `cmd/*/main.go`.
- **SRP**: Every file has a single purpose.
- **Testing**: Use `gotest`. Standard library only. Mocks for all I/O.
- **Docs First**: Update docs before coding. Use `gopush` to publish.
- **WASM Frontend**: Use `tinywasm/fmt` instead of `fmt`/`strings`/`strconv`/`errors`.

## Problem

When TUI runs in **client mode** (connected to daemon via SSE), user actions (Enter on fields, shortcut keys) **never reach the daemon**. The TUI appears frozen — no handler responds.

## Root Cause Analysis

There are **two cascading bugs** in `HeadlessTUI.AddHandler()` ([headless_tui.go:115-132](../headless_tui.go#L115-L132)):

### Bug 1: Handler Type Mismatch — No Handler Matches `execution` or `interactive`

`HeadlessTUI` defines local interfaces for type-switching:

```go
type execution interface {
    Execute()
    Shortcut() string  // ← REQUIRES Shortcut() string
}
type interactive interface {
    Execute(string)
    Shortcut() string  // ← REQUIRES Shortcut() string
}
```

But **real handlers do NOT implement `Shortcut() string`**. They implement `devtui.ShortcutProvider`:

```go
type ShortcutProvider interface {
    Shortcuts() []map[string]string  // plural, returns multiple shortcuts
}
```

**Result**: No handler matches `execution` or `interactive` in the type switch. Handlers with `Value()+Change()` fall to `edit` (no key). Others fall to `display`. All `ch.key` remain empty.

### Bug 2: Edit Handlers Have No Dispatch Key

Even for handlers correctly identified as `edit` ([headless_tui.go:124-126](../headless_tui.go#L124-L126)):

```go
case edit:
    ch.handlerType = htEdit
    ch.action = h.Change
    // ch.key is NEVER set!
```

The serialized `StateEntry.Shortcut` is empty → client-side `postAction()` bails at line 74:

```go
if shortcut == "" || client == nil {
    return  // ← always returns for edit handlers
}
```

And `DispatchAction()` also skips them:

```go
if h.key != "" && h.key == key  // ← h.key is "", never matches
```

### Bug 3: Client Mode Doesn't Register Shortcuts

In client mode, remote fields are created via `newRemoteField()` ([remote_handler.go:14](../../devtui/remote_handler.go#L14)), but **shortcuts are never registered** in the `ShortcutRegistry`. The `registerShortcutsIfSupported()` path only runs for local handler registration. So pressing shortcut keys (e.g., "L" for Large WASM mode) does nothing.

## Solution

### Approach: Use `handler.Name()` as Dispatch Key + `Shortcuts() []map[string]string` as Canonical Interface

The `Shortcut() string` method in HeadlessTUI was an error — it was never the correct interface. The canonical signature is `Shortcuts() []map[string]string` (from `devtui.ShortcutProvider`), which preserves insertion order via ordered single-entry maps. No backward compatibility is needed; we simply adopt the correct interface.

- Use `handler.Name()` as the primary dispatch key for all handler types
- Extract shortcuts from `ShortcutProvider.Shortcuts()` and register as additional dispatch entries
- No new interfaces — only align HeadlessTUI with the existing `ShortcutProvider` contract

---

## Execution Steps

### Stage 1: Fix HeadlessTUI (tinywasm/app)

**File**: [headless_tui.go](../headless_tui.go)

#### 1.1 Remove wrong `execution` and `interactive` local interfaces

Remove the local interfaces that require `Shortcut() string` — this method never existed on any handler. Replace with interfaces matching what handlers actually implement (`Execute()`, `Execute(string)`, `Value()+Change()`).

#### 1.2 Use `Name()` as default dispatch key for all handler types

After the type switch, set `ch.key = handlerName` for ALL handler types that have an action (edit, execution, interactive). This ensures `DispatchAction()` can always route by handler name.

#### 1.3 Extract shortcuts from `Shortcuts() []map[string]string` and register as additional dispatch entries

After capturing the main handler, check if it implements the canonical `Shortcuts() []map[string]string` interface. If so, create additional `capturedHandler` entries for each shortcut key, all pointing to the same `action` function.

#### 1.4 Update `GetHandlerStates()` to include shortcut info from `ShortcutProvider`

The `stateEntry.Shortcut` field should contain the handler's `Name()` (used as the primary dispatch key). Additionally, add a `Shortcuts` field (JSON: `"shortcuts"`) containing the list of shortcut keys from `ShortcutProvider`, so the client can register them.

**New type switch logic**:

```go
type changer interface {
    Value() string
    Change(string)
}
type executor interface {
    Execute()
}
type interactor interface {
    Execute(string)
}
type shortcutProvider interface {
    Shortcuts() []map[string]string
}

switch h := handler.(type) {
case interactor:
    ch.handlerType = htInteractive
    ch.key = handlerName
    ch.action = h.Execute
case executor:
    ch.handlerType = htExecution
    ch.key = handlerName
    ch.action = func(string) { h.Execute() }
case changer:
    ch.handlerType = htEdit
    ch.key = handlerName
    ch.action = h.Change
case display:
    ch.handlerType = htDisplay
}

// Register ShortcutProvider shortcuts as additional dispatch entries
if sp, ok := handler.(shortcutProvider); ok {
    for _, m := range sp.Shortcuts() {
        for key := range m {
            shortcutCh := capturedHandler{
                tabTitle:    tabTitle,
                handlerName: handlerName,
                handlerColor: color,
                handlerType: ch.handlerType,
                key:         key,
                action:      ch.action,
                handler:     handler,
            }
            t.handlers = append(t.handlers, shortcutCh)
        }
    }
}
```

### Stage 2: Update StateEntry Wire Format (tinywasm/devtui)

**File**: [state_entry.go](../../devtui/state_entry.go)

#### 2.1 Add `Shortcuts` field to `StateEntry`

```go
type StateEntry struct {
    // ... existing fields ...
    Shortcut  string              `json:"shortcut"`   // primary key = handler Name()
    Shortcuts []map[string]string `json:"shortcuts"`  // from ShortcutProvider
}
```

#### 2.2 Update `GetHandlerStates()` in HeadlessTUI to populate `Shortcuts`

When serializing state, check if handler implements `shortcutProvider` and include the shortcuts list.

### Stage 3: Register Remote Shortcuts in Client Mode (tinywasm/devtui)

**File**: [remote_handler.go](../../devtui/remote_handler.go)

#### 3.1 Accept `*DevTUI` reference in `newRemoteField()`

Change signature to `newRemoteField(entry StateEntry, client *mcp.Client, ts *tabSection, tui *DevTUI)`.

#### 3.2 Register shortcuts from `StateEntry.Shortcuts` in the `ShortcutRegistry`

After creating the remote field, iterate over `entry.Shortcuts` and register each key in the TUI's `shortcutRegistry`, pointing to the field's tab/field index and using the shortcut key as the value passed to `Change()`.

### Stage 4: Update postAction to Use Handler Name

**File**: [remote_handler.go](../../devtui/remote_handler.go)

#### 4.1 Use `StateEntry.Shortcut` (handler Name) as dispatch key

This already works once HeadlessTUI sets `ch.key = handlerName`. The `postAction` sends `key=handlerName` → daemon's `DispatchAction` matches by `h.key == handlerName` → calls `h.action(value)`.

No code change needed here — it already uses `e.Shortcut` which will now be populated.

### Stage 5: Tests

#### 5.1 HeadlessTUI tests (tinywasm/app)

- **TestHeadlessTUI_EditHandler_DispatchByName**: Register an edit handler, verify `DispatchAction("HandlerName", "value")` calls `Change("value")`.
- **TestHeadlessTUI_ShortcutProvider_DispatchByKey**: Register a handler implementing `ShortcutProvider`, verify `DispatchAction("L", "L")` calls `Change("L")`.
- **TestHeadlessTUI_GetHandlerStates_IncludesShortcuts**: Verify serialized state includes `shortcut` (Name) and `shortcuts` (from ShortcutProvider).

#### 5.2 Remote handler tests (tinywasm/devtui)

- **TestRemoteField_PostsHandlerName**: Verify `postAction` sends handler name as key.
- **TestRemoteField_RegistersShortcuts**: Verify shortcuts from StateEntry are registered in ShortcutRegistry.

## Affected Packages

| Package | Changes | Version Bump |
|---------|---------|-------------|
| `tinywasm/app` | HeadlessTUI type switch + shortcut dispatch | patch |
| `tinywasm/devtui` | StateEntry + remote_handler + shortcut registration | patch |

## Flow After Fix

```
User presses "L" in client TUI
    ↓
shortcutRegistry.Get("L") → entry{Value:"L", HandlerName:"WASM"}
    ↓
executeShortcut() → field.handler.Change("L")
    ↓
postAction(client, "WASM", "L")  ← uses handler Name as key
    ↓
JSON-RPC: tinywasm/action {key:"WASM", value:"L"}
    ↓
HeadlessTUI.DispatchAction("WASM", "L")
    ↓
h.key=="WASM" matches → h.action("L") → handler.Change("L")
    ↓
WasmClient switches to Large mode ✓
```

Alternative flow (shortcut key directly):
```
JSON-RPC: tinywasm/action {key:"L", value:"L"}
    ↓
HeadlessTUI.DispatchAction("L", "L")
    ↓
h.key=="L" matches (shortcut entry) → h.action("L") → handler.Change("L") ✓
```
