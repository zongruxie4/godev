# PLAN: Fix Empty TUI Sections in Client Mode

> Date: 2026-04-08
> Status: Ready to execute
> Scope: `daemon.go`, `bootstrap.go`

---

## Symptom

When `tinywasm` starts normally (daemon + client mode), the TUI shows tabs BUILD, DEPLOY, WIZARD, but the sections appear empty — the frame is visible but no logs or handler controls appear inside them.

---

## Root Cause Analysis

### Architecture Recap

- **Daemon** (`runDaemon`): headless process on port 3030. Runs the project loop. Publishes structured logs via SSE on `/logs`. Exposes state/action endpoints.
- **Client** (`runClient`): real TUI. Connects to daemon SSE stream. Fetches state via `GET /tinywasm/state`. Dispatches actions via `POST /tinywasm/action`.
- **devtui SSE client**: uses `mcp.Client.Call("tinywasm/state", ...)` and `mcp.Client.Dispatch("tinywasm/action", ...)`, which send JSON-RPC to `POST /mcp`.

### The Disconnect

After the MCP API migration, `tinywasm/action` and `tinywasm/state` were moved from JSON-RPC (`RegisterMethod`) to **plain HTTP endpoints**:

| Endpoint | Daemon after migration | What devtui/bootstrap sends |
|---|---|---|
| `POST /mcp` | Only handles MCP protocol (initialize, tools/list, tools/call) | `{"method":"tinywasm/state"}` and `{"method":"tinywasm/action"}` via `mcp.Client` |
| `GET /tinywasm/state` | Returns handler states | — nobody calls this |
| `POST /tinywasm/action` | Dispatches key/value actions | — nobody calls this |

Result: `mcp.Server.HandleMessage` receives `tinywasm/state` and `tinywasm/action`, finds no match, returns `METHOD_NOT_FOUND`. **The project loop in the daemon never starts. No SSE logs are published. Sections remain empty.**

### Bug 1 — devtui calls the wrong endpoint

`devtui/sse_client.go` calls:
```go
// State fetch (on connect and on StateRefresh signal)
h.mcpClient().Call(context.Background(), "tinywasm/state", nil, callback)

// Action dispatch (keyboard shortcuts, remote handler interactions)
mcp.Client.Dispatch("tinywasm/action", &ActionArgs{Key, Value})
```

Both send JSON-RPC to `POST /mcp`. devtui is an **external package** — cannot be modified here.

**Fix location**: `daemon.go` — intercept these methods inside the `POST /mcp` handler before delegating to `mcpServer.HandleMessage`.

### Bug 2 — `bootstrap.go` uses `map[string]string` for params

```go
mcp.NewClient(baseURL, apiKey).Dispatch(twctx.Background(), "tinywasm/action", map[string]string{
    "key":   "start",
    "value": cfg.StartDir,
})
```

`mcp.Client.buildBody` only encodes params if they implement `fmt.Fielder`. `map[string]string` does not. The request is sent with **empty params**, so even if the endpoint were correct, the daemon would receive `key=""` and `value=""` — the "start" command never fires.

**Fix location**: `bootstrap.go` — replace with a plain HTTP POST to `/tinywasm/action`.

---

## Wire Protocol (Confirmed from mcp v0.1.1)

**Request** (sent by `mcp.Client`, tinywasm/json encoding, lowercase keys):
```json
{"jsonrpc":"2.0","id":"1","method":"tinywasm/action","params":"{\"key\":\"start\",\"value\":\"/path\"}"}
```
Note: `params` is a **JSON-encoded string** (double-encoded).

**Response** (expected by `mcp.Client.Call`):
```json
{"jsonrpc":"2.0","id":"1","result":"<json-encoded-string>"}
```
Note: `result` is a **JSON-encoded string** — the call callback receives `[]byte(envelope.result)`.

For `tinywasm/state`, the result string must be a JSON-encoded `[]devtui.StateEntry` array:
```json
{"jsonrpc":"2.0","id":"1","result":"[{\"tab_title\":\"BUILD\",\"handler_name\":\"WasmClient\",...}]"}
```

---

## Changes Required

### File: `daemon.go` — Extend `POST /mcp` handler to intercept custom methods

```go
mux.HandleFunc("POST /mcp", func(w http.ResponseWriter, r *http.Request) {
    var msg []byte
    if r.Body != nil {
        msg, _ = io.ReadAll(r.Body)
    }

    // Extract method from JSON-RPC body to intercept custom app methods.
    // mcp.Server only handles standard MCP protocol; tinywasm/state and
    // tinywasm/action must be intercepted here because devtui calls them
    // via mcp.Client (JSON-RPC to /mcp), not via the plain HTTP endpoints.
    var rpcEnvelope struct {
        ID     string `json:"id"`
        Method string `json:"method"`
        Params string `json:"params"`
    }
    json.Unmarshal(msg, &rpcEnvelope)

    switch rpcEnvelope.Method {
    case "tinywasm/state":
        dtp.mu.Lock()
        projectTui := dtp.projectTui
        dtp.mu.Unlock()

        var stateJSON []byte
        if projectTui != nil {
            stateJSON = projectTui.GetHandlerStates()
        } else {
            stateJSON = ui.GetHandlerStates()
        }

        // result must be a JSON-encoded string (double-encoded) per mcp wire protocol
        resultStr, _ := json.Marshal(string(stateJSON))
        resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":%s}`,
            rpcEnvelope.ID, resultStr)
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(resp))

    case "tinywasm/action":
        var p struct {
            Key   string `json:"key"`
            Value string `json:"value"`
        }
        // params is double-encoded: first un-JSON the outer string
        json.Unmarshal([]byte(rpcEnvelope.Params), &p)

        handled := false
        dtp.mu.Lock()
        projectTui := dtp.projectTui
        dtp.mu.Unlock()
        if projectTui != nil && projectTui.DispatchAction(p.Key, p.Value) {
            handled = true
        } else if ui.DispatchAction(p.Key, p.Value) {
            handled = true
        }
        if !handled {
            switch p.Key {
            case "start":
                if p.Value != "" {
                    logger("Start command received for path:", p.Value)
                    go dtp.startProject(p.Value)
                }
            case "stop":
                dtp.stopProject()
            case "restart":
                dtp.restartCurrentProject()
            default:
                logger("Unknown UI action:", p.Key)
            }
        }

        resultStr, _ := json.Marshal("OK")
        resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":%s}`,
            rpcEnvelope.ID, resultStr)
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(resp))

    default:
        // Standard MCP protocol
        ctx := twctx.Background()
        authHeader := r.Header.Get("Authorization")
        if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
            ctx.Set(mcp.CtxKeyAuthToken, authHeader[7:])
        } else {
            ctx.Set(mcp.CtxKeyAuthToken, authHeader)
        }
        resp := mcpServer.HandleMessage(ctx, msg)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }
})
```

Note: The existing `POST /tinywasm/action` and `GET /tinywasm/state` plain HTTP endpoints can remain as-is for future non-devtui callers. The `/mcp` intercept handles the devtui JSON-RPC path.

### File: `bootstrap.go` — Replace `mcp.Client.Dispatch` with direct HTTP POST

```go
// Before (broken: map[string]string does not implement fmt.Fielder → empty params)
mcp.NewClient(baseURL, apiKey).Dispatch(twctx.Background(), "tinywasm/action", map[string]string{
    "key":   "start",
    "value": cfg.StartDir,
})

// After (correct: plain HTTP POST to the dedicated action endpoint)
import "bytes"
import "net/http"

body, _ := json.Marshal(map[string]string{"key": "start", "value": cfg.StartDir})
req, _ := http.NewRequest("POST", baseURL+"/tinywasm/action", bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
if apiKey != "" {
    req.Header.Set("Authorization", "Bearer "+apiKey)
}
http.DefaultClient.Do(req)
```

This uses the existing `POST /tinywasm/action` plain HTTP endpoint and properly encodes the params with stdlib `encoding/json`.

---

## Result

After both fixes:
1. On client startup, `bootstrap.go` correctly POSTs to `/tinywasm/action` with `{"key":"start","value":"<dir>"}` → daemon starts the project loop
2. devtui's `fetchAndReconstructState` calls `tinywasm/state` via `/mcp` → intercepted, returns `[]StateEntry` → devtui reconstructs remote handler fields in the TUI sections
3. Daemon's headless `Start()` runs `InitBuildHandlers()`, wires `RelayLog`, publishes logs via SSE → devtui `handleLogEvent` routes by `TabTitle` → BUILD/DEPLOY sections display logs

---

## Execution Order

```
1. daemon.go  — extend POST /mcp handler (intercept tinywasm/state + tinywasm/action)
2. bootstrap.go — replace mcp.Client.Dispatch with http.Post
```

Both changes are independent and can be applied in either order. No new files required.
