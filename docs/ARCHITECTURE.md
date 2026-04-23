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
4. **SSR Asset Extraction & Image Optimization (New)**:
   - `app` orchestrates `tinywasm/assetmin` and `tinywasm/imagemin` to extract assets and optimize images from all modules (local and external).
   - **Initial Load**: Discovery happens in background goroutines at startup via `LoadSSRModules()` and `LoadImages()`.
   - **Hot Reload**: `GoModHandler` acts as a relay; when an `ssr.go` file changes, it triggers `OnSSRFileChange` which re-extracts assets and re-processes images without restarting the server.

## 4. MCP Daemon & TUI Client Architecture
**CRITICAL**: Bubble Tea (DevTUI) and MCP both require `stdio`, causing lockups if shared.
**Solution**: The project employs a Persistent Global Daemon architecture.
- **Global Daemon (`tinywasm -mcp`)**: Runs persistently on port `3030` (configurable via `TINYWASM_MCP_PORT`). It uses the `mcp.Server` implementation and native Go `http` routing. It registers global tools like `start_development` (via `daemonToolProvider`) and manages headless project execution, shielding the LLM from restarts.
- **TUI Client (`tinywasm`)**: When a user types `tinywasm`, it detects the daemon on `3030`, runs `app.Start` in `clientMode` (to inject layout sections), and connects strictly as a viewer via Server-Sent Events (`/logs`).
- **Keyboard Webhooks**: In Client Mode, keys like `q` (quit) and `r` (reload) are routed seamlessly via HTTP POST to `http://localhost:3030/tinywasm/action`.

**IDE Configuration**: 
- Transport: `http` (SSE)
- URL: `http://localhost:3030/mcp`
- Config: Automatically managed via `app.ConfigureIDEs` for Cursor and VSCode.

## 5. MCP Tool Availability Timeline

Tools registered with MCP are **async and depend on runtime initialization**:

```timeline
t=0:     start_development RPC called
         └─ daemon spawns runProjectLoop() goroutine
            └─ Start() begins (async)
  
t=~0.5s: onProjectReady(h) callback fires
         ├─ h.WasmClient ✓ (created in AddSectionBUILD)
         ├─ h.Browser ✓ (created in Handler constructor)
         ├─ ProjectToolProxy.SetActive(providers) called
         │  └─ Tools registered: app_rebuild + 16 browser_* tools
         └─ h.OnProjectReady(wg) enqueues background services
  
t=5-10s: Compilation complete, watcher running
         └─ Full project ready, all tools functional
```

**Important**: Tools appear **~500ms after** `start_development` returns, not instantly.
MCP clients (Claude Code, etc.) should:
- Initially call `tools/list` and receive only `start_development`
- Listen for `notifications/tools/list_changed` (SSE) OR poll `tools/list` after 1-2s
- Expect 18 tools once the project is ready

## 6. Startup Flow (`start.go`)
1. Initialize KVDB -> Configure Modes (Local vs Server).
2. Wire `ServerInterface`, `WasmClient`, `AssetMin`, `ImageHandler`, `Goflare`.
3. Configure IDEs for MCP (Port 3030).
4. Concurrently run: HTTP Server (or External Process), DevWatcher, TUI, MCP.
5. Background: Trigger `LoadSSRModules()` and `LoadImages()` to populate the asset cache.
