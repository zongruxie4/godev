# Base A: tinywasm/app — Server Decoupling

> **Status**: Implemented in `docs/SERVER_ARCHITECTURE.md`.

> **Goal**: Remove the hard dependency on `*server.ServerHandler`. Replace it with
> `ServerInterface` + `ServerFactory` so `main.go` decides which concrete server to
> inject. `tinywasm/app` itself does NOT import `tinywasm/server` or `tinywasm/wasi`.

---

## 1. `app/interface.go` — Add ServerInterface and ServerFactory

```go
// ServerInterface is the common contract for all server backends.
// Implemented by: tinywasm/server.ServerHandler, tinywasm/wasi.WasiServer
type ServerInterface interface {
    // Lifecycle
    StartServer(wg *sync.WaitGroup)
    StopServer() error
    RestartServer() error
    // devwatch.FilesEventHandler
    NewFileEvent(fileName, extension, filePath, event string) error
    UnobservedFiles() []string
    SupportedExtensions() []string
    // TUI (devtui.HandlerEdit)
    Name() string
    Label() string
    Value() string
    Change(v string) error
    RefreshUI()
    // Route Registration
    RegisterRoutes(fn func(*http.ServeMux))
}

// ServerFactory creates and configures the concrete server.
// Routes and callbacks are NOT passed here — they are registered directly
// on the returned ServerInterface after InitBuildHandlers creates them.
type ServerFactory func() ServerInterface
```

---

## 2. `app/handler.go` — Change field

```go
// Remove:
ServerHandler *server.ServerHandler

// Add:
Server        ServerInterface
serverFactory ServerFactory
```

All references to `h.ServerHandler` become `h.Server`.

---

## 3. `app/section-build.go` — Use factory + RegisterRoutes

Remove the entire `server.New(&server.Config{...})` block.

```go
func (h *Handler) InitBuildHandlers() {
    // 1. Create server via factory (no routes yet)
    h.Server = h.serverFactory()

    // 2. Create WasmClient and AssetsHandler as before
    h.WasmClient = client.New(...)
    h.AssetsHandler = assetmin.New(...)

    // 3. Register routes directly — same pattern as assetmin itself
    h.Server.RegisterRoutes(h.AssetsHandler.RegisterRoutes)
    h.Server.RegisterRoutes(h.WasmClient.RegisterRoutes)

    // 4. Wire server-specific callbacks via type assertion (server only)
    type externalModeSupport interface {
        SetOnExternalModeExecution(fn func(bool))
    }
    if srv, ok := h.Server.(externalModeSupport); ok {
        srv.SetOnExternalModeExecution(func(isExternal bool) {
            if h.WasmClient != nil {
                h.WasmClient.SetBuildOnDisk(isExternal, true)
            }
            if h.AssetsHandler != nil {
                h.AssetsHandler.SetExternalSSRCompiler(func() error { return nil }, isExternal)
            }
        })
    }

    // 5. Register server with TUI and watcher as before
}
```

---

## 4. `app/handler_lifecycle.go` — Rename references

```go
h.ServerHandler.StartServer(wg)  →  h.Server.StartServer(wg)
h.ServerHandler.Restart()        →  h.Server.RestartServer()
// etc. — global rename ServerHandler → Server
```

---

## 5. `app/start.go` — Add ServerFactory parameter

```go
func Start(
    startDir string,
    logger func(...any),
    tui TuiInterface,
    browser BrowserInterface,
    db DB,
    exitChan chan bool,
    serverFactory ServerFactory,  // ← NEW
    // ... existing variadic
) *Handler
```

Inside `Start()`: `h.serverFactory = serverFactory`.

---

## 6. `app/cmd/tinywasm/main.go` — Factory selection via KVDB

```go
import (
    "github.com/tinywasm/server"
    "github.com/tinywasm/wasi"
)

// ... db initialized above as 'db'

serverType, err := db.Get("TINYWASM_SERVER")
if err != nil {
    serverType = "server" // Default fallback
}

var srv app.ServerFactory
switch serverType {

case "wasi":
    srv = func() app.ServerInterface {
        return wasi.New().
            SetAppRootDir(startDir).
            SetLogger(logger.Logger).
            SetExitChan(exitChan).
            SetUI(ui)
            // Routes added by InitBuildHandlers via RegisterRoutes
    }

default: // "" or "server"
    srv = func() app.ServerInterface {
        s := server.New().
            SetAppRootDir(startDir).
            SetLogger(logger.Logger).
            SetExitChan(exitChan).
            SetStore(db).
            SetUI(ui).
            SetOpenBrowser(browser.OpenBrowser).
            SetGitIgnoreAdd(gitHandler.GitIgnoreAdd).
            SetCompileArgs(func() []string { return []string{"-p", "1"} }).
            SetRunArgs(func() []string {
                args := []string{
                    "-public-dir=" + filepath.Join(startDir, cfg.WebPublicDir()),
                    "-port=" + cfg.ServerPort(),
                }
                if devMode { args = append(args, "-dev") }
                return append(args, wasmClientArgs()...)
            })
        return s
        // Routes added by InitBuildHandlers via RegisterRoutes
        // OnExternalModeExecution wired via type assertion in section-build.go
    }
}

app.Start(startDir, logger.Logger, ui, browser, db, exitChan, srv, ...)
```

---

## 7. `tinywasm/server` — Add Set* methods + RegisterRoutes

`server.New()` becomes zero-arg. Current `Config` fields become Set* methods:

```go
func New() *ServerHandler

func (h *ServerHandler) SetAppRootDir(dir string) *ServerHandler
func (h *ServerHandler) SetSourceDir(dir string) *ServerHandler
func (h *ServerHandler) SetOutputDir(dir string) *ServerHandler
func (h *ServerHandler) SetPublicDir(dir string) *ServerHandler
func (h *ServerHandler) SetMainInputFile(name string) *ServerHandler
func (h *ServerHandler) SetPort(port string) *ServerHandler          // default "6060"
func (h *ServerHandler) SetHTTPS(enabled bool) *ServerHandler
func (h *ServerHandler) SetLogger(fn func(...any)) *ServerHandler    // replaces SetLog
func (h *ServerHandler) SetExitChan(ch chan bool) *ServerHandler
func (h *ServerHandler) SetOpenBrowser(fn func(string, bool)) *ServerHandler
func (h *ServerHandler) SetStore(s Store) *ServerHandler
func (h *ServerHandler) SetUI(ui UI) *ServerHandler
func (h *ServerHandler) SetOnExternalModeExecution(fn func(bool)) *ServerHandler
func (h *ServerHandler) SetGitIgnoreAdd(fn func(string) error) *ServerHandler
func (h *ServerHandler) SetCompileArgs(fn func() []string) *ServerHandler
func (h *ServerHandler) SetRunArgs(fn func() []string) *ServerHandler
func (h *ServerHandler) SetDisableGlobalCleanup(disable bool) *ServerHandler

// Route registration — same pattern as wasi and assetmin:
func (h *ServerHandler) RegisterRoutes(fn func(*http.ServeMux)) *ServerHandler
```

> **WARNING**: This is a **BREAKING CHANGE**. `server.New()` signature changes from `New(*Config)` to `New()`.
> Backward compatibility is **NOT** maintained. The `Config` struct is refactored for internal use only or removed from the public API.

---


## 8. Testing Strategy

Tests **MUST** mock external dependencies using the definitions in `app/test/mock_test.go`. This includes:
- `MockServer` for the server implementation (CRITICAL for decoupling).
- `mockTUI` for UI interactions.
- `MockBrowser` for browser control.
- `MockGitClient` for git operations.
- `MockGitHubClient` for GitHub API interactions.

Example:
```go
// In your test file
import "github.com/tinywasm/app/test"

func TestMyFeature(t *testing.T) {
    mockUI := test.NewUiMockTest()
    // ... use mockUI
}
```

## Modified Files Summary

| File | Change |
|---|---|
| `app/interface.go` | Add `ServerInterface`, `ServerFactory` |
| `app/handler.go` | `ServerHandler *server.ServerHandler` → `Server ServerInterface` + `serverFactory` |
| `app/section-build.go` | Remove `server.New()`, call `h.serverFactory()`, use `RegisterRoutes`, type-assert for `OnExternalModeExecution` |
| `app/handler_lifecycle.go` | Rename `ServerHandler` → `Server` |
| `app/start.go` | Add `serverFactory ServerFactory` parameter |
| `app/cmd/tinywasm/main.go` | Read `TINYWASM_SERVER`, build factory with Set* chain, pass to `Start()` |
| `app/go.mod` | Add `tinywasm/wasi`; keep `tinywasm/server` (both used in `main.go`) |
| `server/server.go` | `New()` zero-arg, add Set* methods, add `RegisterRoutes` |
| `server/go.mod` | No changes |

## Verification

```bash
# Default (server):
cd some-project && tinywasm

# WASI:
# Use .env or appropriate kvdb mechanism

# Tests:
cd tinywasm/app && gotest
```
