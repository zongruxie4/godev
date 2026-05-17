# PLAN: Orchestrate in-memory → disk flush before external server boot

> Status: Ready for execution. Breaking change. No backwards compatibility shims.
> Visual: [diagrams/EXTERNAL_MODE_TRANSITION.md](diagrams/EXTERNAL_MODE_TRANSITION.md) — buggy vs. expected end-to-end flow.

This is the **orchestrator PLAN**. It owns the integration; each affected
library owns its own contract changes in its own PLAN:

- [tinywasm/assetmin/docs/PLAN.md](../../assetmin/docs/PLAN.md) — `FlushToDisk`,
  `EnableSSRMode`, `SetSSRCompiler`, dedupe, snapshot-then-write.
- [tinywasm/server/docs/PLAN.md](../../server/docs/PLAN.md) — replace
  `OnExternalModeExecution` with `BeforeExternalServerStart() error`.
- [tinywasm/client/docs/PLAN.md](../../client/docs/PLAN.md) — replace
  `SetBuildOnDisk(bool, bool)` with `UseDiskStorage` / `UseMemoryStorage`.

## Bug being fixed

When the user switches from internal Go server to external server mode (the build
generates `web/server.go` and a separate process is launched), assets currently held
in memory by `assetmin` are **not** persisted to `web/public/`. The external server
boots against a stale or empty public directory and serves 404s / outdated content.

The defect spans three packages: each has its own contract bug (documented in the
respective PLAN). `app` is the only place that sees the full end-to-end flow and
owns the orchestration.

## Root cause in `tinywasm/app`

[app/section-build.go:69-83](../section-build.go#L69-L83) wires the transition
through the legacy callback `SetOnExternalModeExecution`:

```go
srv.SetOnExternalModeExecution(func(isExternal bool) {
    if h.WasmClient != nil {
        h.WasmClient.SetBuildOnDisk(isExternal, true)
    }
    if h.AssetsHandler != nil {
        h.AssetsHandler.SetExternalSSRCompiler(func() error { return nil }, isExternal)
    }
})
```

Three orchestrator-level defects:

### B1 — Conflated API call at transition time
Passing a no-op `func() error { return nil }` as the "external SSR compiler" is
nonsense; the only intent is to toggle disk mode. The orchestrator is forced into
this shape by `assetmin`'s bad API (fixed in [assetmin PLAN](../../assetmin/docs/PLAN.md)).

### B2 — Asynchronous, non-blocking transition
`OnExternalModeExecution` is fire-and-forget; the server proceeds to
`strategy.Start` without waiting. Race condition. Fixed by the new synchronous
hook in [server PLAN](../../server/docs/PLAN.md). This PLAN consumes that fix.

### B3 — Init-time call conflates SSR-mode activation with a fake compiler
[app/section-build.go:58-60](../section-build.go#L58-L60):

```go
// Activate SSR hot-reload path from startup so CSS changes update the correct
// module-name keyed slot instead of appending a duplicate full-path entry.
h.AssetsHandler.SetExternalSSRCompiler(func() error { return nil }, false)
```

The comment explains the real purpose: turn on the SSR event-handling branch in
`assetmin`. The call achieves this as a side-effect of passing a non-nil function
because `isSSRMode()` is defined as `c.onSSRCompile != nil`. **Not dead code** —
removing it without a replacement breaks CSS hot-reload from the first event.

Fixed by calling `EnableSSRMode()` (new explicit API from
[assetmin PLAN §1](../../assetmin/docs/PLAN.md)).

## Breaking redesign (orchestration only)

### 1. Replace the init-time SSR activation

[app/section-build.go:60](../section-build.go#L60):

```go
// BEFORE (delete):
h.AssetsHandler.SetExternalSSRCompiler(func() error { return nil }, false)

// AFTER:
h.AssetsHandler.EnableSSRMode()
```

No `SetSSRCompiler` call — `app` has no real Go SSR compiler to register.

### 2. Replace the transition wiring

[app/section-build.go:69-83](../section-build.go#L69-L83):

```go
type externalModeSupport interface {
    SetBeforeExternalServerStart(fn func() error) *server.ServerHandler
}
if srv, ok := h.Server.(externalModeSupport); ok {
    srv.SetBeforeExternalServerStart(func() error {
        // 1. Client: switch to disk storage, then compile synchronously.
        //    Must happen BEFORE assetmin flushes because assetmin embeds the
        //    wasm filename (which depends on client mode) into main.js /
        //    index.html via GetSSRClientInitJS(). Dependency is on client
        //    *state*, not disk I/O.
        h.WasmClient.UseDiskStorage()
        if err := h.WasmClient.Compile(); err != nil {
            return fmt.Errorf("wasm compile failed: %w", err)
        }

        // 2. AssetMin: flush ALL in-memory assets to web/public/ (overwrite).
        if err := h.AssetsHandler.FlushToDisk(); err != nil {
            return fmt.Errorf("assetmin flush failed: %w", err)
        }
        return nil
    })
}
```

Notes:
- **One-way transition.** No "back to memory" path; if the user removes
  `web/server.go`, the process restarts cleanly into internal mode.
- **Hook idempotency** is owned by the server contract (see server PLAN). `app`'s
  hook implementation is naturally idempotent: `UseDiskStorage` is idempotent,
  `Compile` is safe to re-run, `FlushToDisk` overwrites.
- **Watcher race.** `Compile()` writes the `.wasm` artifact which can trigger a
  `NewFileEvent` on assetmin from the file watcher. The event may arrive during
  or after `FlushToDisk`. Harmless: assetmin's `diskMirrored` mode after a
  successful flush ensures the late event also propagates to disk.

## Integration tests

Reproducer skeleton already committed at
[../external_mode_flush_test.go](../external_mode_flush_test.go)
(skipped with `t.Skip` until the underlying APIs exist). Coverage:

| Test                                              | What it asserts                                                              |
|---------------------------------------------------|------------------------------------------------------------------------------|
| `TestExternalMode_FlushesAllAssetsBeforeStart`    | Every in-memory asset is on disk *before* `strategy.Start` is invoked.       |
| `TestExternalMode_StartOrderIsSynchronous`        | Recorded order: client.UseDiskStorage → client.Compile → assetmin.FlushToDisk → strategy.Start. |
| `TestExternalMode_FlushErrorAbortsServerStart`    | If `FlushToDisk` returns error, `strategy.Start` is NOT called.              |
| `TestExternalMode_InitEnablesSSRWithoutCompiler`  | After `InitBuildHandlers`, assetmin is in SSR mode with no compiler set.     |
| `TestExternalMode_HookFiresOnEveryExternalStart`  | Hook invoked on each external-mode `StartServer`, not only the first.        |
| `TestExternalMode_RestartDoesNotFireHook`         | `RestartServer` does not invoke the hook.                                    |

These are **integration tests** by design — they verify the composition of three
packages. Unit-level tests of each package's contract live in that package's tree
(see the respective PLANs).

### Test-placement policy (for the implementing agent)

A test belongs to the package whose contract it verifies, not the package that
motivated writing it. Do NOT relocate `server/test/before_external_hook_test.go`,
`client/tests/storage_rename_test.go`, or `assetmin/tests/flush_to_disk_test.go`
into `app/` "for proximity to the bug" — those are unit tests of each library's
own public API and belong with their respective packages. Only the cross-package
integration scenarios above live in `app/`.

## Affected files (this repo, `app/` only)

- [app/section-build.go](../section-build.go) — wiring rewrite (lines 60, 69-83)
  and call-site migration `SetBuildOnDisk` → `UseDiskStorage` + explicit `Compile()`.
- Any other site under `app/` that calls `SetOnExternalModeExecution`,
  `SetExternalSSRCompiler`, `assetmin.SetBuildOnDisk`, or `client.SetBuildOnDisk` —
  grep before changing.
- Tests under `app/` referencing those legacy APIs.

Migrations in other packages (`goflare/`, etc.) are owned by their own packages.

## Execution order for the external agent

1. **`tinywasm/assetmin`** — implement new APIs per
   [assetmin/docs/PLAN.md](../../assetmin/docs/PLAN.md).
2. **`tinywasm/server`** — implement new hook per
   [server/docs/PLAN.md](../../server/docs/PLAN.md).
3. **`tinywasm/client`** — implement storage rename per
   [client/docs/PLAN.md](../../client/docs/PLAN.md).
4. **`tinywasm/app`** — wire all three together per this PLAN.

Each step lands its own tests; integration tests in this PLAN only become
runnable after step 4.

## Acceptance criteria

End-to-end criteria (per-package criteria live in each package's PLAN):

1. [app/section-build.go](../section-build.go) line 60 calls `EnableSSRMode()`
   (NOT deleted — CSS hot-reload depends on the SSR event branch being active
   from startup).
2. [app/section-build.go](../section-build.go) lines 69-83 are replaced by the
   `BeforeExternalServerStart` wiring shown in §2.
3. The six integration tests pass.
4. Manual repro: with a dev session running, create `web/server.go`, observe
   that `web/public/main.css`, `main.js`, `index.html`, sprite.svg and every
   module CSS exist and match what the in-memory dev server was serving.

## Out of scope

- Watcher / Browser reload pipeline.
- Deploy handlers.
- Anything inside `assetmin`, `server`, or `client` — owned by their own PLANs.
