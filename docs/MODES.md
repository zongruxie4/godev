# TinyWasm Execution Modes

TinyWasm `app` operates in two primary execution modes: **In-Memory Mode** and **External Server (Disk) Mode**. Understanding these modes is crucial for debugging and refactoring.

## 1. In-Memory Mode (Default / Rapid Prototyping)

This mode is active when:
- No `server.go` file exists in the server source directory.
- `server_external_mode` is set to `false` in configuration/database.

### Characteristics:
- **No Compilation**: The server logic is part of the `tinywasm/app` process itself.
- **Virtual File System**: It serves files from memory or directly from the source directory without an intermediate build step for the server backend.
- **Limited Backend**: You cannot add custom backend logic (routes, handlers) easily in this mode, as it uses a default internal server.
- **Fast Startup**: Ideal for frontend-only development or initial layout (maquetación).

## 2. External Server Mode (Disk Mode)

This mode is active when:
- A `server.go` file exists (or is created) in the server source directory.
- `server_external_mode` is set to `true`.
- You explicitly switch to "External Execution" in the TUI or config.

### Characteristics:
- **Full Compilation**: The server code (e.g., `web/server.go`) is compiled into a binary (e.g., `web/server` or `web/server.exe`).
- **Separate Process**: The compiled server runs as a separate process, managed by `tinywasm/app`.
- **Custom Backend**: You have full control over the Go `net/http` server, allowing custom routes, middleware, and logic.
- **Hot Reload**: When `server.go` or dependencies change, `tinywasm/app` stops the old process, recompiles, and starts the new one.
- **Argument Propagation**: Arguments (like `-usetinygo`, `-port`) are passed from `app` to the external process.

### Mode Switching & Synchronization

When switching implementation details (like Go -> TinyGo for WASM), the system must ensure synchronization:

1. **WASM Compilation**: `tinywasm/app` (via `WasmClient`) recompiles the WASM binary.
2. **Flag Propagation**: The new mode (e.g., `size mode: S`) is passed to the server process arguments strings (e.g. `-wasmsize_mode=S`).
3. **Server Restart**: If running in **External Mode**, `tinywasm/app` **MUST** restart the external server process so it receives the new arguments.
4. **Asset Serving**: The restarted server initializes `tinywasm/site` with the new flags, ensuring `client.Javascript` serves the correct `wasm_exec.js` glue code matching the new WASM binary.

> **Troubleshooting Tip**: If you see `WebAssembly.instantiate` errors about missing imports (like `wasi_snapshot_preview1`), it usually means the server is serving the wrong `wasm_exec.js` (Go vs TinyGo) because it wasn't restarted or didn't receive the correct mode flags.
