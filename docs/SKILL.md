---
name: TinyWasm App Architecture & Troubleshooting
description: Guide to understanding the internals of tinywasm/app, its execution modes, and debugging common issues.
---

# TinyWasm App Architecture & Troubleshooting

This skill provides context for refactoring and debugging `tinywasm/app`.

## Core Components

- **App Handler (`app.Handler`)**: The central orchestrator. Initializes all other handlers.
- **Server Handler (`server.ServerHandler`)**: Manages the backend server.
    - Supports **In-Memory** (internal) and **External** (compiled binary) strategies.
    - **Crucial**: Must be restarted via `Restart()` when runtime arguments change (e.g. WASM mode switch).
- **Wasm Client (`client.WasmClient`)**: Manages WASM compilation.
    - Handles Go (Standard) and TinyGo compilations.
    - Generates `wasm_exec.js` and `client.wasm`.
- **AssetMin (`assetmin.AssetMin`)**: Manages asset minification and injection (CSS/JS).
- **Site (`site`)**: The library used by the *user's* server implementation to mount handlers.

## Execution Flows

### Build & Watch Flow
1. **Selection**: User selects "BUILD" section.
2. **Watch**: `devwatch` detects file changes.
3. **Rebuild**:
    - If `.go` (backend) changes -> `ServerHandler.handleFileEvent` -> Restart Server.
    - If `.go` (frontend/wasm) changes -> `WasmClient` recompiles -> `OnWasmExecChange` callback -> Restart Server (to update args) -> Reload Browser.

### Mode Propagation
- `wasmsize_mode` (S, M, L) determines if TinyGo is used.
- This mode is passed to the *External Server* via command-line flags (`-wasmsize_mode=S`).
- The external server (using `tinywasm/site`) parses this flag using `client.NewJavascriptFromArgs()`.
- `site` configures the correct `wasm_exec.js` based on the parsed mode.

## Common Issues & Fixes

### 1. `WebAssembly.instantiate` TypeError / Import Error
**Symptoms**: Browser console shows error about `wasi_snapshot_preview1` or module not being an object.
**Cause**: Mismatch between the served `wasm_exec.js` and the compiled `client.wasm`.
- Scenario A: WASM compiled with TinyGo, but server sends Go's `wasm_exec.js`.
- Scenario B: Server process wasn't restarted after mode switch, so it's using old flags.
**Fix**:
- Ensure `ServerHandler.Restart()` is called when WASM mode changes.
- Ensure `ArgumentsForServer` in `client.go` includes updated flags.
- Check `tinywasm/site` logic in the user's `server.go` (or `mount.back.go`) uses `NewJavascriptFromArgs`.

### 2. "Undefined" errors during development
**Cause**: Missing `replace` directives in `go.mod` when modifying multiple interdependent local modules (`app`, `client`, `site`).
**Fix**: Add `replace github.com/tinywasm/pkg => ../pkg` in the relevant `go.mod`.

## Reference
- [Execution Modes](./MODES.md) - Detailed info on Memory vs Disk mode.
- `tinywasm/app/section-build.go` - Main build orchestration logic.
