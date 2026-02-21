# `tinywasm/app` Architecture & Guide (LLM Context)

`tinywasm/app` is the central orchestrator and developer CLI (`tinywasm`). It manages the build pipeline, file watcher, external servers, TUI, and MCP.

## 1. Project Structure (User Projects)
All apps managed by `tinywasm/app` **MUST** follow this structure:
```
projectName/
├── .env, .gitignore, go.mod
├── modules/          # Business logic (not importable). Modules must implement Init() []any
├── pkg/              # Shared code (safe to import)
└── web/              # Frontend & Backend logic
    ├── client.go     # Web client (//go:build wasm)
    ├── server.go     # Go server (//go:build !wasm)
    ├── public/       # Static resources (HTML, CSS, JS, WASM, images)
    └── ui/           # Visual components
```

## 2. Execution Modes & Server Decoupling
The app operates in two primary modes based on the presence of user's `web/server.go`:

### A. In-Memory Mode (Internal Server)
- Used when `server.go` is missing or `server_external_mode=false`.
- Fast, auto-generates `web/client.go` if empty.
- No custom backend logic possible; serves static/WASM via interior defaults.

### B. External Server Mode
- Used when `server.go` exists and `server_external_mode=true`.
- Compiles the user's `server.go` to a binary (`web/server`) and runs it as a child process.
- **Server Decoupling**: `app` uses `app.ServerInterface` and `app.ServerFactory`.
  - `main.go` reads config and injects the concrete server (default `server.ServerHandler` or `wasi.WasiServer`). 
  - `InitBuildHandlers()` registers routes (`assetmin` and `WasmClient`) directly into the injected server via `RegisterRoutes()`.

## 3. DevWatch & Build Pipeline
`tinywasm/devwatch` orchestrates the rebuilds when files change:
1. **Frontend Change (`.go` in WASM paths, or `web/ui`)**:
   - `WasmClient` recompiles `client.wasm` (using Go or TinyGo based on mode `S/M/L`).
   - If using External Server, **the server MUST be restarted** to receive updated flags (e.g., `-wasmsize_mode`).
   - Reloads browser via `devbrowser`.
2. **Backend Change (`.go` server files)**:
   - Restarts the external server process.
3. **WASI Builder (Optional)**:
   - Watches `modules/*/wasm/`, compiles generic `.wasm` via `tinygo -target wasi`, hot-swaps payloads.

## 4. MCP vs TUI (STDIO Conflict)
**CRITICAL**: Bubble Tea (DevTUI) and MCP both require `stdio`, causing lockups if shared.
**Solution**: MCP **MUST** run over HTTP via `StreamableHTTP`, natively on port `3100` (configurable via `--mcp-port`).
- Standard startup: `tinywasm` (starts TUI on stdio, MCP on `:3100`).
- Claude config: Transport `http`, URL `http://localhost:3100/mcp`.

## 5. Startup Flow (`start.go`)
1. Initialize KVDB -> Configure Modes (Local vs Server).
2. Wire `ServerInterface`, `WasmClient`, `AssetMin`, `Goflare`.
3. Configure VS Code MCP (Port 3100).
4. Concurrently run: HTTP Server (or External Process), DevWatcher, TUI, MCP.
