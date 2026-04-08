# Stage 2 — Remove sseHubAdapter

### Context
`*sse.SSEServer` satisfies `mcp.SSETransport` directly in the new `tinywasm/mcp` API. The `sseHubAdapter` is no longer needed.

### Affected Files
- `sse_adapter.go` — delete everything except `logChannelProvider` if it is still used for generic SSE.

### Required Changes in `daemon.go`
```go
// Before
sseHub := &sseHubAdapter{tinySSE.Server(&sse.ServerConfig{...})}
mcpHandler := mcp.NewHandler(mcpConfig, sseHub, toolProviders)

// After
sseServer := tinySSE.Server(&sse.ServerConfig{...})
// SSE is passed via Config
```

### Steps
- [ ] Remove `sseHubAdapter.Publish()` wrapper.
- [ ] Remove `sseHubAdapter` struct entirely.
- [ ] Retain `logChannelProvider` in `sse_adapter.go` (or rename the file).
- [ ] Update `daemon.go` construction to use `sseServer` directly in `mcp.Config`.
