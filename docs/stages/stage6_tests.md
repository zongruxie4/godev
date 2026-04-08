# Stage 6 — Integration Tests

### Minimum Required Suite (Post-Migration)

We need robust black-box tests ensuring the server launches correctly and binds tools correctly using `WasmClient`.

```
TestDaemon_MCPServer_StartsWithValidAuth
    → daemon boots, MCPServer initializes smoothly without error.

TestDaemon_MCPServer_NilAuth_Panics
    → if auth is nil, it must fail cleanly on boot, not during runtime execution.

TestApp_ToolProvider_Tools_NotEmpty
    → Handler.Tools() contains at least `app_rebuild`.

TestProjectProxy_SetActive_UpdatesTools
    → SetActive replacing an old provider. Tools() must reflect the swap on next call.

TestMCPTool_AppRebuild_Execute_CallsWasmClient
    → Calling `app_rebuild` through `mcp.Execute` invokes `WasmClient.RecompileMainWasm`.

TestMCPTool_AppRebuild_NoWasmClient_ReturnsError
    → If `WasmClient` is nil, Execute fails returning descriptive text (no panics).

TestHTTPHandler_MCP_Endpoint_Reachable
    → `POST /mcp` returning JSON-RPC success for `ping`.

TestHTTPHandler_MCP_InvalidToken_401
    → `POST /mcp` returning -32001 or standard fail for invalid JSON-RPC payload.

TestSSEServer_PublishesOnToolCall
    → Successful execution publishes via SSE Hub Adapter or equivalent HTTP transport.
```

### Steps
- Create `tests/mcp_migration_test.go` with all listed cases.
- Apply mocks for `WasmClient` and `SSETransport`.
- Use `-race` flag for concurrency validations during HTTP vs JSON RPC handling.
