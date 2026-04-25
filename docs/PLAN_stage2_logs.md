# PLAN Stage 2 — Log file elimination & MCP tab

## Goal
Remove the always-created `logs.log` file from project directories.
Normal logs flow exclusively through TUI/SSE. File output is reserved for internal tinywasm errors, only when `--debug` is active.
Add a dedicated MCP tab so the developer can inspect daemon status without generating files.

## Problem
`logs.go:Logger()` writes to `logs.log` every time it is called once `initialized == true`.
The `--debug` flag exists in `main.go` but is never forwarded to the `Logger` struct.
This pollutes every Go project with a `logs.log` that belongs to tinywasm, not the user project.

## Dependency
None. This stage is self-contained within `tinywasm/app`.

---

## Change 1 — `logs.go`: split Logger from InternalError

### Before
```go
func (l *Logger) Logger(messages ...any) {
    if !l.initialized { return }
    // always writes to logs.log
}
```

### After
```go
// Logger sends messages through the normal channel (TUI/SSE). Never writes to file.
func (l *Logger) Logger(messages ...any) { /* no-op file write */ }

// InternalError writes to logs.log only when debug mode is active.
// Use only for errors internal to tinywasm itself, not for project build errors.
func (l *Logger) InternalError(messages ...any) {
    if !l.debug { return }
    // write to logs.log at RootDir
}
```

Add `debug bool` field to `Logger` struct.
Add `func (l *Logger) SetDebug(v bool)` method.

### Files
| File | Change |
|------|--------|
| [logs.go](../logs.go) | Add `debug bool` field; rename file-write logic to `InternalError`; `Logger` no longer writes to file |
| [cmd/tinywasm/main.go](../cmd/tinywasm/main.go) | Add `logger.SetDebug(*debugFlag)` after `logger.SetRootDir(...)` |

---

## Change 2 — Replace all `logger.Logger` error call sites with `logger.InternalError`

Audit every call to `logger.Logger(...)` in `cmd/tinywasm/main.go` that represents a tinywasm internal failure (git init error, kvdb failure, etc.) and replace with `logger.InternalError(...)`.
Project-level log calls (build output, server events) stay as `logger.Logger(...)`.

### Files
| File | Change |
|------|--------|
| [cmd/tinywasm/main.go](../cmd/tinywasm/main.go) | Lines 40, 58: change `logger.Logger(...)` → `logger.InternalError(...)` for tinywasm-internal errors |

---

## Change 3 — `sse_publisher.go`: add ring buffer for `app_get_logs`

The SSE hub history is internal to `tinywasm/sse` (unexported). `SSEPublisher` is the single relay point for all logs — add a 100-entry ring buffer there so `app_get_logs` can read recent entries without a file.

```go
type SSEPublisher struct {
    hub    ssePublisher
    mu     sync.Mutex
    ring   [100]string
    head   int
    count  int
}

func (p *SSEPublisher) addToRing(msg string) { /* circular write */ }
func (p *SSEPublisher) RecentLogs() []string { /* returns up to 100 in order */ }
```

Call `p.addToRing(msg)` inside `PublishTabLog`.

### Files
| File | Change |
|------|--------|
| [sse_publisher.go](../sse_publisher.go) | Add ring buffer fields + `addToRing` + `RecentLogs()` |

---

## Change 4 — `daemon.go`: `app_get_logs` reads ring buffer, not file

Replace `executeGetLogs` implementation: read from `d.ssePub.RecentLogs()` instead of `os.ReadFile(logPath)`.
Note: the `lastPath` field remains in the struct — it is still used by `startProject` (line 528)
and `executeGetLogs`'s path resolution (lines 436, 514). Only the file-read logic inside
`executeGetLogs` changes.

### Files
| File | Change |
|------|--------|
| [daemon.go](../daemon.go) | `executeGetLogs`: replace file read with `d.ssePub.RecentLogs()` |

---

## Change 5 — `section-mcp.go`: new MCP tab (BUILD → DEPLOY → MCP)

Create `section-mcp.go` with `AddSectionMCP()`. Move `h.MCP` `AddHandler` call from `start.go` to this new section.

```go
// section-mcp.go
func (h *Handler) AddSectionMCP() any {
    section := h.Tui.NewTabSection("MCP", "MCP Daemon Status")
    h.SectionMCP = section
    return section
}
```

In `start.go`, there are two changes:

1. Add `h.SectionMCP = h.AddSectionMCP()` after `h.AddSectionDEPLOY()` and **before** the `clientMode` early-return guard (currently after line 70, before line 78). This matches how BUILD and DEPLOY are registered — all three sections must exist before `clientMode` exits so the client TUI can reconstruct them.

2. Where `h.Tui.AddHandler(h.MCP, colorOrangeLight, h.SectionBuild)` is called (currently standalone mode block, search by context not line number since earlier changes shift lines), change `h.SectionBuild` → `h.SectionMCP`.

Do not use line numbers to locate the second change — search for the `colorOrangeLight` + `h.MCP` combination.

Add `SectionMCP any` field to `Handler` struct in `handler.go`.

### Files
| File | Change |
|------|--------|
| [section-mcp.go](../section-mcp.go) | New file: `AddSectionMCP()` |
| [handler.go](../handler.go) | Add `SectionMCP any` field |
| [start.go](../start.go) | Add `AddSectionMCP()` call; move MCP `AddHandler` to `h.SectionMCP` |

---

## Tests

### Test 1 — `Logger` never writes file (logs_test.go)
```
- Create Logger with tmpDir
- Call Logger.Logger("anything")
- Assert logs.log does NOT exist in tmpDir
```

### Test 2 — `InternalError` writes file only when debug=true (logs_test.go)
```
- debug=false: call InternalError → file must NOT exist
- debug=true:  call InternalError → file must exist and contain the message
```

### Test 3 — `SSEPublisher.RecentLogs` ring buffer (sse_publisher_test.go)
```
- Publish 150 entries
- RecentLogs() must return exactly 100, in order, last 100 entries
```

### Test 4 — `app_get_logs` returns ring content, not file (daemon_test.go or inline)
```
- Call PublishTabLog 5 times via ssePub
- Call executeGetLogs
- Assert result contains those 5 messages (no file on disk)
```

## Steps

- [ ] `logs.go`: add `debug` field + `SetDebug`; move file write to `InternalError`; `Logger` becomes TUI-only
- [ ] `cmd/tinywasm/main.go`: `logger.SetDebug(*debugFlag)`; replace internal error calls with `logger.InternalError`
- [ ] `sse_publisher.go`: add ring buffer + `RecentLogs()`
- [ ] `daemon.go`: `executeGetLogs` reads `d.ssePub.RecentLogs()`
- [ ] `section-mcp.go`: create new file
- [ ] `handler.go`: add `SectionMCP any`
- [ ] `start.go`: wire `AddSectionMCP()` and move MCP `AddHandler`
- [ ] Write tests 1–4
- [ ] Run `gotest` — all must pass
- [ ] Verify `logs.log` is NOT created in a clean project run (smoke test)
- [ ] Verify LLM `app_get_logs` tool returns recent log lines without any file on disk
