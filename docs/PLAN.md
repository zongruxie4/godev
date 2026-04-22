# PLAN: MCP tools not visible to Claude Code after dynamic registration

## Problem

Claude Code connects to the tinywasm MCP server at startup and receives the initial `tools/list`.
After `start_development` fires and project tools are registered dynamically (via `mcp.Server.AddTool`
+ `notifications/tools/list_changed` SSE notification), Claude Code **does not reload the tool list**.

**Result:** Claude Code only sees the tools available at the moment of its initial connection.
If the project was already running when Claude Code connected, it sees all tools correctly.
If Claude Code connected before/during project startup, it misses dynamically added tools.

**Confirmed via direct HTTP query:**
```
POST http://localhost:42425/mcp  →  tools/list
Returns 14 tools: browser_*, start_development, app_get_logs
```
But Claude Code's tool list shows 0 tools (only the MCP server entry in the UI, no callable tools).

## Root cause analysis

There are two separate issues:

### Issue 1 — Claude Code does not re-fetch tools/list on SSE notification
The MCP spec (2024-11-05) defines `notifications/tools/list_changed` as a hint that the client
should re-fetch `tools/list`. Claude Code appears to not implement this — it does not re-fetch
after receiving the notification.

### Issue 2 — Claude Code uses HTTP transport, not SSE stream
The tinywasm MCP is configured as `"type": "http"` in `~/.claude/settings.json`.
With HTTP transport, notifications sent via SSE (`/logs` channel) may not reach Claude Code
because it polls `/mcp` via POST only and doesn't maintain a persistent SSE connection.

## Current configuration (`~/.claude/settings.json`)
```json
"tinywasm": {
  "type": "http",
  "url": "http://localhost:42425/mcp"
}
```

## Solutions

### Option A — Pre-register all tools at server startup (recommended short-term)
Register `browser_*` and `app_*` tools at daemon startup, before any project starts.
Tools that have no active project simply return a friendly error ("no project running").

**Pros:** Works with any MCP client regardless of notification support.
**Cons:** Tools appear even when no project is active (minor UX issue).

**Implementation:**
- In `daemon.go`, register all known tools in `newDaemonToolProvider.Tools()` upfront
- Each tool's `Execute` checks `d.toolProxy` for an active project; returns error if none
- Remove dynamic `AddTool` call in `onProjectReady` (tools already registered)

### Option B — Add `app_get_logs` and `app_reload_tools` tools
Expose a tool `app_reload_tools` that Claude Code can call manually to force a re-discovery.
Also expose `app_get_logs` to read build/runtime logs without needing SSE.

**Pros:** Gives LLM a recovery path when tools are missing.
**Cons:** Requires user/LLM to know to call it.

### Option C — Investigate SSE transport for Claude Code
Change MCP config from `"type": "http"` to `"type": "sse"` if Claude Code supports it,
to receive `notifications/tools/list_changed` on a persistent connection.

**Note:** Claude Code currently supports `http` and `stdio` transports. SSE as a separate
transport type may not be supported — needs verification.

## Action items

- [x] **A1** — Moved browser_* tools to `daemonToolProvider.Tools()` — pre-registered at daemon startup
- [x] **A2** — Each browser tool's Execute delegates via `executeBrowserTool()` → `toolProxy.Tools()` → returns friendly error if no active project
- [x] **A3** — Added `app_get_logs` to `daemonToolProvider.Tools()` — reads `lastPath/logs.log` directly
- [x] **A4** — Removed dynamic `AddTool` loop in `onProjectReady` — no longer needed
- [ ] **B1** — `app_reload_tools` tool not implemented — Option A makes it unnecessary
- [ ] **C1** — Test `"type": "sse"` transport in Claude Code settings — deferred

## Affected files

- `app/daemon.go` — `newDaemonToolProvider`, `onProjectReady`, `buildProjectProviders`
- `app/mcp-tools.go` — `Handler.Tools()` (currently returns empty slice)
- `app/mcp_registry.go` — `buildProjectProviders`
- `~/.claude/settings.json` — MCP transport config
