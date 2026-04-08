# Stage 3 — Migrate Config and Routing

### Context
`tinywasm/mcp` removed `Port` and `APIKey` from `Config` (they moved to `app`), and explicitly requires an `HTTPEngine` to mount routes. Also, security is required.

### `daemon.go` Changes

```go
// Before
mcpConfig := mcp.Config{ Port, ServerName, ServerVersion, AppName, AppVersion }
mcpHandler := mcp.NewHandler(...)
mcpHandler.SetAuth(mcp.NewTokenAuthorizer(apiKey))
// ...
http.Handle("/mcp", mcpHandler.HTTPHandler())

// After
var auth mcp.Authorizer
if apiKey != "" {
    auth = mcp.NewTokenAuthorizer(apiKey)
} else {
    auth = mcp.OpenAuthorizer() // explicit opt-in
}

mcpServer, err := mcp.NewServer(mcp.Config{
    Name:    "TinyWasm - Global MCP Server",
    Version: cfg.Version,
    Auth:    auth,
    SSE:     sseServer,
}, toolProviders)

if err != nil {
    fmt.Printf("MCP Init failed: %v", err)
    os.Exit(1)
}

// mcpServer uses RegisterRoutes on our App mux
mcpServer.RegisterRoutes(tinySSE.GetMux()) // Or whichever mux the app uses

// IDE Config moved out of MCP! Done via `app.ConfigureIDEs`
if err := mcpIDE.Configure(cfg.AppName, cfg.Version, mcpPort, apiKey); err != nil {
    // log ...
}
```

### Steps
- [ ] Update `daemon.go` — new `mcp.Config` usage.
- [ ] Call `mcpServer.RegisterRoutes(mux)` to correctly mount endpoints.
- [ ] Update `start.go` equivalent.
- [ ] Replace `MCP *mcp.Handler` with `MCP *mcp.Server` inside `app.Handler`.
