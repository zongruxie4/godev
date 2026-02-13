# Server Architecture

> **Goal**: The server component is decoupled from the main application logic (`tinywasm/app`) using dependency injection via `ServerInterface` and `ServerFactory`.

---

## Core Components

### 1. ServerInterface (`app/interface.go`)

The `ServerInterface` defines the contract that any server backend must implement. It includes methods for lifecycle management, file watching, TUI integration, and route registration.

```go
type ServerInterface interface {
    // Lifecycle
    StartServer(wg *sync.WaitGroup)
    StopServer() error
    RestartServer() error
    // ...
    RegisterRoutes(fn func(*http.ServeMux))
}
```

### 2. ServerFactory (`app/interface.go`)

The `ServerFactory` is a function type that creates and returns a `ServerInterface` implementation. It allows `main.go` to decide which concrete server implementation to use.

```go
type ServerFactory func() ServerInterface
```

### 3. ServerWrapper (`cmd/tinywasm/main.go`)

Since the concrete `server.ServerHandler` implementation (from `github.com/tinywasm/server`) returns `*ServerHandler` in some methods (like `RegisterRoutes` and configuration setters), it does not directly satisfy the `ServerInterface` (which expects void return for `RegisterRoutes`) or the configuration interfaces used in `app` package.

To bridge this gap, `main.go` implements a `ServerWrapper` struct that embeds `*server.ServerHandler` and adapts the method signatures by hiding return values.

```go
type ServerWrapper struct {
    *server.ServerHandler
}

func (w *ServerWrapper) RegisterRoutes(fn func(*http.ServeMux)) {
    w.ServerHandler.RegisterRoutes(fn)
}

// Setters implemented with void return to satisfy type assertions in app package
func (w *ServerWrapper) SetRunArgs(fn func() []string) { w.ServerHandler.SetRunArgs(fn) }
// ...
```

---

## Configuration Flow

1. **Instantiation**: `cmd/tinywasm/main.go` reads configuration (e.g., from DB) to select the server implementation (defaulting to `server`, optionally `wasi`). It creates a `ServerFactory` closure that instantiates the server (wrapped in `ServerWrapper`).

2. **Injection**: The `ServerFactory` is passed to `app.Start()`.

3. **Creation**: `app.Handler.InitBuildHandlers` calls the factory to create the server instance: `h.Server = h.serverFactory()`.

4. **Routing**: Routes from `AssetMin` and `WasmClient` are registered via `h.Server.RegisterRoutes`.

5. **Advanced Configuration**: The `app` package configures server-specific settings (like run arguments, compile arguments, directories) using type assertion on interfaces defined locally in `section-build.go`.

```go
    // In app/section-build.go
    type serverConfigurator interface {
        SetRunArgs(func() []string)
        // ...
    }

    if srv, ok := h.Server.(serverConfigurator); ok {
        srv.SetRunArgs(...)
    }
```

This pattern ensures `tinywasm/app` does not have a hard dependency on `github.com/tinywasm/server`, allowing alternative implementations (like `wasi`) to be swapped in easily.
