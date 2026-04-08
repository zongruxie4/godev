# Stage 5 — IDE Configuration Migration

### Context
In previous unreleased versions, `tinywasm/mcp` had a method `ConfigureIDEs()` tightly coupled to the MCP library itself. For proper separation of concerns (avoiding environment reading loops, and keeping `mcp` stateless), this side-effect orchestration was moved to `tinywasm/app`.

### Migration Logic
1. **Move Code**: Extract `ide_config.go` (or `handler_ide.go` logic) out of `tinywasm/mcp`, into `tinywasm/app/mcp_ide/` or similar.
2. **Rewrite Configuration Runner**: Create `ConfigureIDEs(appName, port, apiKey string)` on the app level.
3. **No MCP config bindings**: This operation is entirely decoupled from `mcp.Config`. It runs directly as a daemon boot phase right before or after starting the HTTP server.

### Required Tests (Recovered logic from former MCP)
We must restore the critical coverage tests that were briefly living in `mcp`:

1. `TestConfigureIDEs_WritesVSCodeConfig` 
   - Uses a temporary directory for `$HOME/.config/Code/User`.
   - Bootstraps IDE config.
   - Verifies `mcp.json` has `http://localhost:3030/mcp` + `Bearer token` present.

2. `TestConfigureIDEs_PreservesOtherServers` 
   - Injects a fake VSCode payload containing `"postgres": {...}`.
   - Bootstraps `tinywasm` MCP server.
   - Resulting JSON must contain both `"postgres"` and `"tinywasm"`.

3. `TestConfigureIDEs_UpdatesExistingURL` 
   - Starts with `"tinywasm": { "url": "http://localhost:8080/mcp" }`
   - Config runs with `port = 9000`
   - Resulting JSON must have `"http://localhost:9000/mcp"`.

4. `TestConfigureIDEs_HandlesEmptyPortGracefully`
   - Doesn't crash if port is not yet bound (returns immediately or defaults to internal default port).

### Steps
- [ ] Create `mcp_ide` package under `app/`.
- [ ] Migrate `IDEInfo` and `getVSCodeConfigPath()`, `getAntigravityConfigPath()`, `getClaudeCodeConfigPath()` logic into `mcp_ide/config.go`.
- [ ] Recover and adapt tests inside `mcp_ide/config_test.go`.
- [ ] Call `mcp_ide.Configure(cfg.AppName, cfg.Port, apiKey)` inside `daemon.go` directly.
