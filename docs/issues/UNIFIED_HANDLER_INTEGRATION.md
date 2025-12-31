# Unified Handler Integration

## Objective

Update all handlers in this package and external packages to implement the new `Loggable` interface and remove logger injection via Config.

## Prerequisites

Complete the DevTUI refactoring first (see `tinywasm/devtui/docs/issues/UNIFIED_HANDLER_ARCHITECTURE.md`).

## Pattern: Before and After

### Before (Logger via Config)

```go
// Config receives logger
type Config struct {
    Logger func(message ...any)
}

// Handler uses logger from config
type WasmClient struct {
    config *Config
}

func (w *WasmClient) Compile() {
    w.config.Logger("Compiling...")
}

// In app: create logger and pass to config
wasmLogger := tui.AddLogger("WASM", true, color, tab)
wasmClient := client.New(&client.Config{
    Logger: wasmLogger,
})
```

### After (Loggable Interface)

```go
// Config without logger
type Config struct {
    // Logger removed
}

// Handler has internal log
type WasmClient struct {
    log func(message ...any)
}

// Initialize with no-op
func New(config *Config) *WasmClient {
    return &WasmClient{
        log: func(message ...any) {},
    }
}

// Handler implements Loggable
func (w *WasmClient) Name() string { return "WASM" }

func (w *WasmClient) SetLog(logger func(message ...any)) {
    w.log = logger
}

func (w *WasmClient) Compile() {
    w.log("Compiling...")
}

// In app: just register handler
wasmClient := client.New(&client.Config{})
tui.AddHandler(wasmClient, 0, color, tab)  // DevTUI calls SetLog automatically
```

## Files to Update

### External Packages (breaking changes)

Each package needs:
1. Remove `Logger` from Config
2. Add `log func(message ...any)` field to main struct
3. Initialize `log` as no-op in constructor
4. Add `Name() string` method
5. Add `SetLog(func(message ...any))` method

| Package | Main Struct | Name() Value |
|---------|-------------|--------------|
| `client` | `WasmClient` | `"WASM"` |
| `server` | `Server` | `"SERVER"` |
| `assetmin` | `AssetMin` | `"ASSETS"` |
| `devwatch` | `Watcher` | `"WATCH"` |
| `devbrowser` | `DevBrowser` | `"BROWSER"` |

### This Package (tinywasm/app)

### File: `section-build.go`

Update `AddSectionBUILD`:

```go
func (h *handler) AddSectionBUILD() {
    sectionBuild := h.tui.NewTabSection("BUILD", "Building and Compiling")

    // Create handlers (without logger in config)
    h.wasmClient = client.New(&client.Config{
        SourceDir: h.config.CmdWebClientDir(),
        OutputDir: h.config.WebPublicDir(),
        Database:  h.db,
        // Logger removed
    })

    h.assetsHandler = assetmin.NewAssetMin(&assetmin.Config{
        OutputDir: filepath.Join(h.rootDir, h.config.WebPublicDir()),
        // Logger removed
    })

    h.serverHandler = server.New(&server.Config{
        // Logger removed
    })

    h.browser = devbrowser.New(h.config, h.tui, h.db, h.exitChan)

    h.watcher = devwatch.New(&devwatch.WatchConfig{
        // Logger removed
    })

    // Register all handlers with DevTUI (this calls SetLog automatically)
    h.tui.AddHandler(h.wasmClient, 0, colorPurpleMedium, sectionBuild)
    h.tui.AddHandler(h.serverHandler, 0, colorBlueMedium, sectionBuild)
    h.tui.AddHandler(h.assetsHandler, 0, colorGreenMedium, sectionBuild)
    h.tui.AddHandler(h.watcher, 0, colorYellowMedium, sectionBuild)
    h.tui.AddHandler(h.browser, 0, colorPinkMedium, sectionBuild)

    // ... rest of setup
}
```

## Verification

```bash
go build ./...
go test ./... -v
```

## Completion Checklist

### External Packages
- [ ] `client/client.go`: Implement `Loggable`, remove Logger from Config
- [ ] `server/server.go`: Implement `Loggable`, remove Logger from Config
- [ ] `assetmin/assetmin.go`: Implement `Loggable`, remove Logger from Config
- [ ] `devwatch/watcher.go`: Implement `Loggable`, remove Logger from Config
- [ ] `devbrowser/browser.go`: Implement `Loggable`, remove Logger from Config

### This Package
- [ ] `section-build.go`: Remove wasmLogger, serverLogger, etc.
- [ ] `section-build.go`: Remove Logger from all Config structs
- [ ] `section-build.go`: Use `AddHandler` for all handlers
- [ ] Code compiles without errors
- [ ] All tests pass
