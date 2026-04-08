# Stage 4 — Migrate Tool Definitions & Provider Protocol

### Context
`mcp.Tool` structurally changed. The generic `map[string]any` execution was replaced by strongly typed structs mapped by `.Schema()`. `mcp.ToolProvider` changed its method from `GetMCPTools()` to `Tools()`.

### Change to `Tool` definition
```go
// Old Tool structure
mcp.Tool{
    Name:        "app_rebuild",
    Description: "...",
    Parameters:  []mcp.Parameter{},
    Execute: func(args map[string]any) { ... },
}

// New Tool structure implementation
mcp.Tool{
    Name:        "app_rebuild",
    Description: "...",
    InputSchema: new(mcp.EmptyArgs).Schema(), // or a specific ORM struct
    Resource:    "app",    // RBAC mandatory
    Action:      'u',      // RBAC mandatory
    Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
        // Run logic
        return mcp.Text("rebuild completed"), nil
    },
}
```

### Steps
- [ ] Define argument structs for each existing tool (using `ormc gen` if they take args).
- [ ] For parameterless tools, use `mcp.EmptyArgs{}` and its schema.
- [ ] Assign `Resource` and `Action` based on app RBAC levels.
- [ ] Migrate `Execute` to receive `(ctx *context.Context, req mcp.Request)` returning `(*mcp.Result, error)`.
- [ ] Use `req.Bind(&args)` to parse arguments gracefully.
- [ ] Rename `GetMCPTools()` to `Tools()` across all implementers (e.g. `mcp-tools.go`, `mcp_registry.go`, `daemon.go`).
