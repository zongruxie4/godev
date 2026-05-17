# External server mode transition — flow

Companion to [app/docs/PLAN.md](../PLAN.md) and
[assetmin/docs/PLAN.md](../../../assetmin/docs/PLAN.md).

## Current (buggy) flow

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant App as tinywasm/app<br/>(Handler)
    participant Server as tinywasm/server<br/>(ServerHandler)
    participant Client as tinywasm/client<br/>(WasmClient)
    participant Asset as tinywasm/assetmin<br/>(AssetMin)
    participant Disk as web/public/
    participant Ext as External<br/>server process

    User->>App: create web/server.go
    App->>Server: StartServer(wg)
    Server->>Server: detect web/server.go<br/>switch to external strategy
    Server-)App: OnExternalModeExecution(true)<br/>(fire-and-forget, no error)

    par async — race
        App->>Client: UseDiskStorage + Compile
        Client->>Disk: write .wasm
    and
        App->>Asset: SetExternalSSRCompiler(noop, true)
        Note over Asset: only 5 hardcoded handlers
        Asset->>Disk: FileWriteSafe(main.css) — SKIPPED if exists ❌
        Asset->>Disk: FileWriteSafe(main.js)  — SKIPPED if exists ❌
        Note over Asset,Disk: per-module CSS,<br/>extra SVGs, etc.<br/>NEVER flushed ❌
    and
        Server->>Ext: strategy.Start() — boots immediately
        Ext->>Disk: read web/public/* — 404s / stale ❌
    end
```

Defects highlighted (❌):
- **B1** `FileWriteSafe` skips stale files → in-memory bytes lost.
- **B2** Only 5 main handlers enumerated → module assets never reach disk.
- **B2 (race)** `strategy.Start` runs in parallel with the flush.

## Expected (post-fix) flow

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant App as tinywasm/app<br/>(Handler)
    participant Server as tinywasm/server<br/>(ServerHandler)
    participant Client as tinywasm/client<br/>(WasmClient)
    participant Asset as tinywasm/assetmin<br/>(AssetMin)
    participant Disk as web/public/
    participant Ext as External<br/>server process

    User->>App: create web/server.go
    App->>Server: StartServer(wg)
    Server->>Server: detect web/server.go<br/>switch to external strategy

    Note over Server,App: SYNCHRONOUS HOOK
    Server->>App: BeforeExternalServerStart() error

    activate App
    App->>Client: UseDiskStorage + Compile
    Client->>Disk: write .wasm + wasm_exec.js
    Client-->>App: ok

    App->>Asset: FlushToDisk()
    activate Asset
    loop every registered asset (N, not 5)
        Asset->>Disk: FileWrite(outputPath)<br/>OVERWRITE ✅
    end
    Asset-->>App: nil (or error → abort)
    deactivate Asset

    App-->>Server: nil
    deactivate App

    Note over Asset: disk-mirrored mode ON<br/>future mutations propagate

    Server->>Ext: strategy.Start()
    Ext->>Disk: read web/public/* — complete & fresh ✅
```

Guarantees:
- ✅ `BeforeExternalServerStart` returns BEFORE `strategy.Start` is called.
- ✅ Every in-memory asset is on disk (overwritten, not skipped).
- ✅ A non-nil error from the hook aborts the transition with a logged message;
  `strategy.Start` is never invoked.
- ✅ After flush, `assetmin` mirrors subsequent in-memory mutations to disk.

## Component contracts (post-fix)

```mermaid
classDiagram
    class AssetMin {
        +EnableSSRMode()
        +SetSSRCompiler(fn func() error)
        +FlushToDisk() error
        -ssrEnabled bool
        -onSSRCompile func() error
        -diskMirrored bool
        -allAssets map~string~asset
    }
    class ServerHandler {
        +SetBeforeExternalServerStart(fn func() error)
        -BeforeExternalServerStart func() error
    }
    class Handler {
        -WasmClient
        -AssetsHandler AssetMin
        -Server ServerHandler
        +InitBuildHandlers()
    }
    Handler --> ServerHandler : SetBeforeExternalServerStart(...)
    Handler --> AssetMin : FlushToDisk()
    Handler --> WasmClient : UseDiskStorage + Compile
    ServerHandler ..> Handler : invokes hook synchronously
```

## Removed APIs (no shims)

| Package   | Removed                                          | Replaced by                                  |
|-----------|--------------------------------------------------|----------------------------------------------|
| client    | `SetBuildOnDisk(onDisk, compileNow bool)`        | `UseDiskStorage()` + `UseMemoryStorage()` + explicit `Compile()` |
| assetmin  | `SetExternalSSRCompiler(fn, bool)`               | `EnableSSRMode()` + `SetSSRCompiler(fn)` + `FlushToDisk()` |
| assetmin  | `SetBuildOnDisk(bool)` (deprecated alias)        | `FlushToDisk()`                              |
| assetmin  | `isSSRMode = (onSSRCompile != nil)`              | `c.ssrEnabled` set by `EnableSSRMode`        |
| assetmin  | compiler auto-invoked on registration            | `SetSSRCompiler` is a pure setter            |
| assetmin  | `FileWriteSafe(...)`                             | `FileWrite(...)` (overwrite is correct)      |
| assetmin  | `buildOnDisk` field                              | `diskMirrored` internal flag (set by flush)  |
| server    | `OnExternalModeExecution(isExternal bool)`       | `BeforeExternalServerStart() error`          |
| server    | `SetOnExternalModeExecution(fn)`                 | `SetBeforeExternalServerStart(fn)`           |
