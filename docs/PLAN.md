# PLAN: TUI Input Section Disappears After MCP start_development

## Status: PENDING

## Affected Libraries
- **Primary (root cause)**: `tinywasm/app` — `daemon.go`
- **Secondary (resilience gap)**: `tinywasm/devtui` — `sse_client.go`

---

## Symptom

When `mcp__tinywasm__start_development` is called while the user has the inputs section open
in the real TUI (`tinywasm` client mode), the input section **disappears and never comes back**.
The user must stop and restart `tinywasm` to recover the interface.

---

## Root Cause — `tinywasm/app/daemon.go`

### The Race Condition in `runProjectLoop`

`runProjectLoop` is a goroutine that manages one project's lifetime:

```go
func (d *daemonToolProvider) runProjectLoop(ctx, projectPath string, cancel chan bool) {
    headlessTui := NewHeadlessTUI(d.logger)

    d.mu.Lock()
    d.projectTui = headlessTui   // (A) set new TUI — runs immediately
    d.mu.Unlock()

    defer func() {
        d.mu.Lock()
        d.projectTui = nil        // (B) BUG: always clears, no ownership check
        d.toolProxy.SetActive()   // (B) BUG: also clears proxy unconditionally
        d.mu.Unlock()
    }()

    Start(...)  // blocking — project runs here
}
```

When `start_development` is called while a project is running:

1. `startProject` closes the **old** project's `projectCancel` channel.
2. Waits for port to free (old project's external server stops).
3. Starts **new** goroutine → new goroutine executes **(A)**: sets `d.projectTui = newTUI`.
4. Old goroutine's `defer` eventually runs **(B)**: blindly sets `d.projectTui = nil`.

**Result**: `d.projectTui` becomes `nil` even though the new project is alive and running.

### Why The Inputs Disappear Immediately

The new project's `onProjectReady` callback calls `d.ssePub.PublishStateRefresh()`,
which triggers `devtui.fetchAndReconstructState()`. By the time this fires:

- If the old goroutine's defer **(B)** has already run → `d.projectTui == nil`
- `/tinywasm/state` returns the outer `ui.GetHandlerStates()` (empty daemon TUI) → `[]`
- `devtui` calls `clearRemoteHandlers()` → inputs are gone

Even if the race goes the other way (new goroutine wins, state looks correct), any subsequent
`StateRefresh` event that fires after **(B)** runs will return empty state and wipe the inputs.

### Sequence Diagram

```
startProject()
  ├─ close(oldCancel)
  ├─ wait for port free
  │    (old Start() shuts down → external server stops → port free)
  └─ go runProjectLoop(newPath)  [new goroutine]
       └─ (A) d.projectTui = newTUI

(old goroutine cleanup eventually runs)
  └─ (B) d.projectTui = nil     ← CONTAMINATES new project's state

onProjectReady fires (new project initialized)
  └─ PublishStateRefresh()
       └─ devtui: fetchAndReconstructState()
            └─ GET /tinywasm/state → d.projectTui == nil → returns []
                 └─ clearRemoteHandlers()  ← INPUTS GONE ✗
```

---

## Secondary Issue — `tinywasm/devtui/sse_client.go`

`fetchAndReconstructState` clears all remote handlers when state is empty:

```go
func (h *DevTUI) fetchAndReconstructState() {
    h.mcpClient().Call(..., "tinywasm/state", nil, func(result []byte, err error) {
        if err != nil || result == nil { return }
        var entries []StateEntry
        if err := json.Unmarshal(result, &entries); err != nil { return }

        h.clearRemoteHandlers()               // ← clears even if entries == []
        h.reconstructRemoteHandlers(entries)  // ← adds nothing if entries is empty
        h.RefreshUI()
    })
}
```

An empty state response (transient race or legitimate "no project") causes `clearRemoteHandlers()`
to wipe the inputs with no recovery path until another non-empty `StateRefresh` fires.

---

## Fix

### Fix 1 — `tinywasm/app/daemon.go` (primary fix)

In `runProjectLoop`, add an **ownership check** to the defer so the old goroutine only clears
`projectTui` (and the proxy) if it is still the current project:

```go
defer func() {
    d.mu.Lock()
    if d.projectTui == headlessTui {   // only clear if WE are still the owner
        d.projectTui = nil
        d.toolProxy.SetActive()        // clear proxy only together with tui
    }
    d.mu.Unlock()
}()
```

This is safe because `d.projectTui` is set under the same mutex, so identity comparison
is a reliable ownership signal.

### Fix 2 — `tinywasm/devtui/sse_client.go` (resilience fix)

Skip state updates when the result is empty, so transient nil states don't wipe the UI:

```go
func (h *DevTUI) fetchAndReconstructState() {
    h.mcpClient().Call(..., "tinywasm/state", nil, func(result []byte, err error) {
        if err != nil || result == nil { return }
        var entries []StateEntry
        if err := json.Unmarshal(result, &entries); err != nil { return }
        if len(entries) == 0 { return }   // FIX: don't clear on empty — wait for non-empty refresh

        h.clearRemoteHandlers()
        h.reconstructRemoteHandlers(entries)
        h.RefreshUI()
    })
}
```

This makes devtui resilient: a transient empty state (due to any race or future bug) does not
wipe the currently visible inputs. The next non-empty StateRefresh correctly rebuilds them.

---

## Files to Modify

| File | Change |
|------|--------|
| `tinywasm/app/daemon.go` | `runProjectLoop` defer: add `if d.projectTui == headlessTui` ownership check |
| `tinywasm/devtui/sse_client.go` | `fetchAndReconstructState`: skip update when `entries` is empty |

---

## Tests

| Library | File | Test | Failure before fix |
|---------|------|------|--------------------|
| `tinywasm/app` | `bug_tui_contamination_test.go` | `TestDaemon_OldGoroutineCleanup_DoesNotContaminateNewProjectTUI` | FAIL — old defer wipes new TUI |
| `tinywasm/app` | `bug_tui_contamination_test.go` | `TestDaemon_OldGoroutineCleanup_DoesNotClearProxyAfterNewProjectSet` | FAIL — proxy cleared by old defer |
| `tinywasm/devtui` | `sse_client_empty_state_test.go` | `TestFetchAndReconstructState_EmptyResponseDoesNotClearHandlers` | FAIL — clearRemoteHandlers on empty |

---

## Why It Wasn't Caught Earlier

The existing test `mcp_daemon_test.go` tests proxy behavior but not the lifecycle race
between old goroutine cleanup and new project initialization. The contamination only manifests
when `start_development` is called on an already-running project — a flow not covered by
existing tests, and one that requires real timing between goroutines.
