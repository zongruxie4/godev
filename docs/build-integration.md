# Base 4: tinywasm/app — Build Pipeline Integration

## Changes to `section-build.go`

### 4.1 New handler: `WasiModuleBuilder`
A handler (similar to WasmClient) that:
1. Detects `modules/*/wasm/` directories
2. Compiles each with TinyGo: `tinygo build -target wasi -o output/modules/NAME.wasm ./modules/NAME/wasm/`
3. On file change (`.go` files in `modules/NAME/wasm/`): recompile → server detects new `.wasm` → hot-swap
4. Registered in devwatch as a file handler

```go
// New component in InitBuildHandlers()
h.WasiBuilder = wasibuilder.New(&wasibuilder.Config{
    AppRootDir: rootDir,
    ModulesDir: "modules",
    OutputDir:  filepath.Join(outputDir, "modules"),
    Logger:     logger,
})
```

### 4.2 devwatch integration
```go
// Add to Watcher setup in InitBuildHandlers()
cfg.FilesEventHandlers = append(cfg.FilesEventHandlers, h.WasiBuilder)
```

### 4.3 Server config update
```go
// In server.New() config:
WasiModulesDir:   "modules",
WasiOutputDir:    filepath.Join(outputDir, "modules"),
WasiDrainTimeout: 5 * time.Second,
```

## New optional package: `tinywasm/wasibuilder`
Handles TinyGo WASI compilation (similar to how `gobuild` handles server compilation).
Decision: add to `devflow` as a new cmd or to a new `tinywasm/wasibuilder` package.
