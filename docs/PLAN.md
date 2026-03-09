# PLAN: Add GetLog() to App Handler

## Related Docs
- Depends on: [tinywasm/mcp PLAN.md](../../mcp/docs/PLAN.md) — must be completed first.
- [diagrams/CLIENT_LIFECYCLE.md](diagrams/CLIENT_LIFECYCLE.md)

---

## Development Rules
- No external libraries; standard library only.
- Use `gotest` to run tests. Use `gopush` to publish.

---

## Context

After `tinywasm/mcp` adds `GetLog()` to the `Loggable` interface, any app struct that implements `Loggable` must add `GetLog()`. No tool `Execute` closures need modification.

---

## Step 1 — Identify Loggable implementors

Check which structs implement `Loggable` (have `Name()` + `SetLog()`). Add `GetLog()` to each.

```go
func (h *Handler) GetLog() func(message ...any) {
    return h.log
}
```

---

## Publish
After tests pass: `gopush 'feat: implement GetLog for MCP Loggable interface'`
