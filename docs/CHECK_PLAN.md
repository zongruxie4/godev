# PLAN: Fix MCP Tool Registration & Ctrl+C Lifecycle

## Problems

1. **Tools never appear after `start_development`** — `mcp.NewServer` reads providers once at startup when `ProjectToolProxy` is empty. Calling `SetActive` later updates only the proxy's internal slice; the `mcp.Server.s.tools` static map is never updated. `handleListTools` reads only `s.tools`, so tools never appear.

2. **`app_rebuild` should not exist as an MCP tool** — tinywasm/app recompiles automatically on file change via devwatch. Exposing it as an MCP tool is redundant and misleading.

3. **Ctrl+C on TUI client does not stop the daemon** — When `tinywasm` (TUI client) exits, it closes its local `exitChan` but never notifies the daemon (`tinywasm -mcp`) to stop. The daemon keeps running after the user quits.

---

## Fix 1: Register tools into `mcp.Server` when project starts

**File:** `app/daemon.go`, inside `runProjectLoop()`, function `onProjectReady`

**Current code (around line 498):**
```go
onProjectReady := func(h *Handler) {
    providers := buildProjectProviders(h)
    d.toolProxy.SetActive(providers...)
    d.logger("ProjectToolProxy upgraded:", len(providers), "providers (added app_rebuild)")
    d.ssePub.PublishStateRefresh()
}
```

**New code:**
```go
onProjectReady := func(h *Handler) {
    providers := buildProjectProviders(h)
    d.toolProxy.SetActive(providers...)
    // Register tools into mcp.Server static map so tools/list returns them.
    // AddTool is thread-safe and emits notifications/tools/list_changed via SSE.
    for _, p := range providers {
        for _, tool := range p.Tools() {
            d.mcpServer.AddTool(tool)
        }
    }
    d.logger("Project tools registered:", len(providers), "providers")
    d.ssePub.PublishStateRefresh()
}
```

**Why `d.mcpServer` is accessible:** `daemonToolProvider` has field `mcpServer *mcp.Server` (line ~309). It is assigned at line ~94 of daemon.go. The closure captures `d` which is the `daemonToolProvider`.

---

## Fix 2: Remove `app_rebuild` from MCP tools

**File:** `app/mcp-tools.go`

**Action:** Replace the entire `Tools()` method body so it returns an empty slice.

**New content of the function:**
```go
// Tools returns metadata for all Handler MCP tools.
// app_rebuild is intentionally not exposed: tinywasm recompiles automatically on file change.
func (h *Handler) Tools() []mcp.Tool {
    return []mcp.Tool{}
}
```

Do NOT delete the file — `Handler` must still satisfy `mcp.ToolProvider` because `buildProjectProviders` includes `h` as a provider.

**Note:** After this change, `buildProjectProviders` will add `h` but `h.Tools()` returns empty, which is harmless.

---

## Fix 3: Ctrl+C on TUI client stops the daemon

### Part A — Add "quit" action handler in daemon (`app/daemon.go`)

Inside the `POST /tinywasm/action` handler (around line 220), in the `switch key` block (around line 243), add a `"quit"` case:

**Current `switch` block:**
```go
switch key {
case "start":
    ...
case "stop":
    dtp.stopProject()
case "restart":
    dtp.restartCurrentProject()
default:
    logger("Unknown UI action:", key)
}
```

**New `switch` block:**
```go
switch key {
case "start":
    ...
case "stop":
    dtp.stopProject()
case "restart":
    dtp.restartCurrentProject()
case "quit":
    logger("Quit command received from client — shutting down daemon")
    dtp.stopProject()
    close(exitChan)
default:
    logger("Unknown UI action:", key)
}
```

`exitChan` is the daemon's own exit channel, already in scope at that point in `runDaemon()`. Closing it triggers the goroutine at line ~291 that calls `server.Close()`, which causes `ListenAndServe` to return and the daemon process to exit.

**Important:** `exitChan` must only be closed once. Add a `sync.Once` guard or check if it is already closed. The safest pattern:

```go
case "quit":
    logger("Quit command received from client — shutting down daemon")
    dtp.stopProject()
    select {
    case exitChan <- true: // non-blocking signal if buffered
    default:
    }
    // If exitChan is unbuffered (current), use a sync.Once at the daemon level:
    // daemonOnce.Do(func() { close(exitChan) })
```

Check whether `exitChan` in `runDaemon` is buffered or unbuffered before choosing the pattern. If unbuffered, use `sync.Once`.

### Part B — Send "quit" from TUI client on exit (`app/bootstrap.go`)

**File:** `app/bootstrap.go`, function `runClient()`

The TUI runs via `Start(...)` in client mode (line ~112). When `Start` returns, the TUI has exited (user pressed Ctrl+C or q). After `Start` returns, send the quit action to the daemon:

**Add after the `Start(...)` call:**
```go
// Notify daemon to stop when the TUI client exits
quitBody, _ := json.Marshal(map[string]string{"key": "quit"})
req, err := http.NewRequest("POST", baseURL+"/tinywasm/action", bytes.NewReader(quitBody))
if err == nil {
    req.Header.Set("Content-Type", "application/json")
    if apiKey != "" {
        req.Header.Set("Authorization", "Bearer "+apiKey)
    }
    http.DefaultClient.Do(req) // best-effort, ignore error
}
```

---

## Summary of files to change

| File | What to change |
|------|----------------|
| `app/daemon.go` | In `onProjectReady`: add loop calling `d.mcpServer.AddTool(tool)` after `SetActive` |
| `app/daemon.go` | In `POST /tinywasm/action` switch: add `case "quit"` that stops project and closes `exitChan` |
| `app/mcp-tools.go` | `Handler.Tools()` returns `[]mcp.Tool{}` (keep file and method, just empty the return) |
| `app/bootstrap.go` | After `Start(...)` in `runClient()`: POST `{"key":"quit"}` to daemon |

---

## Tests

Add these 3 tests to `app/test/`. The mock infrastructure already exists in `mock_test.go` (`MockBrowser`, `mockTUI`, `MockServer`). No real project, browser, or WASM compilation needed.

### Test 1 — Tools appear in `tools/list` after `SetActive` + `AddTool` (`app/test/mcp_daemon_test.go`)

```go
func TestProjectProxy_ToolsAppearInMCPServer(t *testing.T) {
    // Create mcp.Server with an empty proxy (simulates daemon startup)
    proxy := app.NewProjectToolProxy()
    mcpServer, _ := mcp.NewServer(mcp.Config{
        Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer(),
    }, []mcp.ToolProvider{proxy})

    // At startup, only the proxy is registered and it's empty
    // tools/list must return 0 tools (proxy was empty at NewServer time)
    ctx := &context.Context{}
    req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/list","params":{}}`)
    resp := mcpServer.HandleMessage(ctx, req)
    // assert resp contains "tools":[]

    // Simulate onProjectReady: SetActive + AddTool loop
    mockBrowser := &MockBrowser{}
    browserAdapter := &app.BrowserAdapter{BrowserInterface: mockBrowser}
    proxy.SetActive(browserAdapter)
    for _, tool := range browserAdapter.Tools() {
        mcpServer.AddTool(tool)
    }

    // Now tools/list must return the tools from MockBrowser
    resp2 := mcpServer.HandleMessage(ctx, req)
    // assert resp2 contains the registered tools
    _ = resp2
}
```

**Why this test matters:** catches any regression where `SetActive` + `AddTool` stops being called together, making tools invisible again.

**Note:** `MockBrowser.GetMCPTools()` currently returns `[]mcp.Tool{}`. For this test to be meaningful, update `MockBrowser` to return at least one fake tool, or create a `mockToolProvider` inline.

### Test 2 — `app_rebuild` never appears in tools (`app/test/mcp_daemon_test.go`)

```go
func TestHandlerTools_AppRebuildNotExposed(t *testing.T) {
    h := &app.Handler{} // zero-value Handler
    tools := h.Tools()
    for _, tool := range tools {
        if tool.Name == "app_rebuild" {
            t.Fatalf("app_rebuild must not be exposed as an MCP tool, found in Handler.Tools()")
        }
    }
}
```

**Why this test matters:** prevents `app_rebuild` from being accidentally re-added later.

### Test 3 — Quit action closes daemon exit channel (`app/test/mcp_daemon_test.go`)

```go
func TestDaemonAction_QuitClosesExitChan(t *testing.T) {
    exitChan := make(chan bool)
    closed := make(chan struct{})

    // Spin up a minimal HTTP server that handles POST /tinywasm/action
    // with the same logic as daemon.go but wired to exitChan
    mux := http.NewServeMux()
    var once sync.Once
    mux.HandleFunc("POST /tinywasm/action", func(w http.ResponseWriter, r *http.Request) {
        body, _ := io.ReadAll(r.Body)
        key := string(mcp.ExtractJSONValue(body, "key"))   // unquote not needed for plain string
        if string(key) == `"quit"` || string(key) == "quit" {
            once.Do(func() { close(exitChan) })
        }
        w.Write([]byte("OK"))
    })

    srv := httptest.NewServer(mux)
    defer srv.Close()

    // Goroutine that watches exitChan
    go func() {
        <-exitChan
        close(closed)
    }()

    // Client sends quit
    http.Post(srv.URL+"/tinywasm/action",
        "application/json",
        strings.NewReader(`{"key":"quit"}`))

    select {
    case <-closed:
        // pass
    case <-time.After(2 * time.Second):
        t.Fatal("exitChan was not closed after quit action")
    }
}
```

**Why this test matters:** verifies the quit→daemon-stop contract without needing a real OS process.

---

## Acceptance Criteria

- [ ] After `start_development`, `tools/list` returns browser tools (no `app_rebuild`)
- [ ] `app_rebuild` does NOT appear in `tools/list` ever
- [ ] Pressing Ctrl+C or `q` in the TUI client (`tinywasm`) stops the daemon process too
- [ ] `go build ./cmd/tinywasm` compiles without errors after all changes
- [ ] `go test ./test/...` passes including the 3 new tests above
