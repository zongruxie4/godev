# PLAN — Hot-reload misses `ssr.go` in sibling subpackages of the project module

## Problem

A reproducer test lives at
[`app/test/hotreload_subpackage_test.go`](../test/hotreload_subpackage_test.go)
(`TestWatcher_IncludesModuleRoot_NotJustStartDir`). It builds a temp
`parent/{web,sub}` layout, calls `InitBuildHandlers` with `startDir =
parent/web`, and asserts the resulting `*devwatch.DevWatch` watches both
`parent/web` (start dir) and `parent` (module root).

Today the test fails with:

```
watcher must include the Go module root ".../001" so ssr.go in
sibling subpackages emits FS events; watched dirs: [.../001/web]
```

After the fix below, it passes.

When the active project is, e.g., `layout/platformd/web/` (a wasm `main`
package whose Go module root lives several directories up at
`layout/go.mod`), editing `layout/platformd/ssr.go` does **not** trigger
SSR re-extraction. The browser keeps serving the previous CSS/HTML/JS
even though the source changed and `go build ./...` shows the change is
valid.

Verified manually:
1. Edit `layout/platformd/ssr.go` (add a new `Rule(...)`).
2. `go build ./...` succeeds (compiler sees the change).
3. `curl -s http://localhost:6060/style.css | grep <new-rule>` → no match.
4. No log lines about "SSR hot reload" appear in the daemon output.

## Root Cause

The hot-reload chain is wired correctly in
[`section-build.go:194-216`](../section-build.go#L194-L216):

```
devwatch FS event
  → GoModHandler.NewFileEvent(...)
    → if fileName == "ssr.go" → OnSSRFileChange(dir)
      → AssetsHandler.ReloadSSRModule(dir)
        → Browser.Reload()
```

But the **watcher's set of observed directories** never includes the
sibling subpackages of the active module:

- [`section-build.go:219`](../section-build.go#L219) →
  `h.Watcher.AddDirectoriesToWatch(h.Config.RootDir)`
  where `h.Config.RootDir` = `startDir` = the project's `web/` dir
  (passed by the user when starting tinywasm on that project).
- [`section-build.go:222-231`](../section-build.go#L222-L231) → adds the
  paths of any `replace` directives in the project's `go.mod`.

Concretely, for `layout/platformd/web/`:
- `startDir = layout/platformd/web/` — only `web/` is watched.
- The Go module root (`layout/`, where `go.mod` lives) is computed in
  [`start.go:42`](../start.go#L42) (`devflow.FindProjectRoot(startDir)`)
  and handed to `GoModHandler.SetRootDir(...)`, but it is **never
  registered with the watcher**.
- `layout/go.mod` declares no `replace` directives, so the second branch
  adds nothing.

Result: events in `layout/platformd/ssr.go` (sibling of the project
subpackage, but inside the same Go module) are invisible to devwatch.
`GoModHandler.NewFileEvent` is never invoked → `OnSSRFileChange` never
fires → `assetmin`'s SSR cache is never invalidated.

This is a hot-reload coverage bug owned by the **orchestrator**
(`tinywasm/app`) — it is the layer that decides which directories
devwatch must observe. `assetmin`'s and `devwatch`'s contracts are both
honoured: assetmin will re-extract on demand; devwatch will fire events
for any directory it has been told to watch.

## Fix

Register the **module root** (where the project's `go.mod` lives) with
the watcher in addition to the start dir. This guarantees that every
file inside the active Go module — including `ssr.go` files in sibling
subpackages — produces FS events.

### Patch sketch — `app/section-build.go` (around line 219)

```go
// Add main project root to watcher
h.Watcher.AddDirectoriesToWatch(h.Config.RootDir)

// Also watch the Go module root so sibling subpackages of startDir
// (e.g. layout/platformd/ when startDir is layout/platformd/web/)
// produce FS events. Without this, ssr.go changes outside startDir
// never reach GoModHandler.NewFileEvent.
if moduleRoot, err := devflow.FindProjectRoot(h.Config.RootDir); err == nil &&
    moduleRoot != "" && moduleRoot != h.Config.RootDir {
    h.Watcher.Logger("WATCH", "Watching Go module root:", moduleRoot)
    h.Watcher.AddDirectoriesToWatch(moduleRoot)
}

// Add local replace modules to watcher automatically
...existing code...
```

Notes:
- Adding `moduleRoot` is idempotent — `AddDirectoriesToWatch` deduplicates.
- When `startDir == moduleRoot` (project at the module root), the second
  call is a no-op.
- `UnobservedFiles` already excludes `.git`, `_test.go`, `.log`, etc., so
  walking the wider tree does not balloon the event set.
- The change does not affect WASM rebuild logic — it only widens the
  observed surface for `GoModHandler.NewFileEvent`.

### Test plan

Regression test:
[`app/test/hotreload_subpackage_test.go`](../test/hotreload_subpackage_test.go) —
`TestWatcher_IncludesModuleRoot_NotJustStartDir`.

It is a structural assertion on the watcher state (not an end-to-end FS
event test) for two reasons:
- Determinism: no fsnotify timing, no goroutine sleeps, no flakiness.
- Localised fault domain: the bug is "the watcher is never told about
  the module root", so checking the watcher's `directories` set is the
  most direct evidence.

The test reads the unexported `*devwatch.DevWatch.directories` field via
reflection+unsafe. devwatch ships no public accessor and adding one is
out of scope here. If devwatch later adds e.g. `Directories() []string`,
swap the helper in the test.

Status: **fails today** (proves the bug), **must pass** after the patch.

## Files to change

- [`app/section-build.go`](../section-build.go) — register module root
  with watcher (~5 lines around line 219).
- [`app/test/hotreload_subpackage_test.go`](../test/hotreload_subpackage_test.go) —
  regression test (already in place; fails today, must pass after fix).

## Verification

1. Build: `go build ./...` from `app/` — passes.
2. Restart tinywasm on `layout/platformd/web/`.
3. Touch `layout/platformd/ssr.go` (add a no-op rule).
4. Daemon logs show:
   `WATCH Watching Go module root: /home/cesar/Dev/Project/tinywasm/layout`
   followed by
   `SSR hot reload …` (from `AssetsHandler.ReloadSSRModule`).
5. `curl -s http://localhost:6060/style.css | grep <new-rule>` → match.

## Out of scope

- Auditing what other handlers (besides `GoModHandler`) might rely on
  the narrow watch set. The widening only adds events; existing
  handlers already filter by extension/name in their `NewFileEvent`,
  so this should be safe — but a quick read of `WasmClient`,
  `AssetsHandler`, and `Server` `NewFileEvent` implementations is
  recommended to confirm none assume "events only come from startDir".
- Improvements to `assetmin`'s `loadSSRModulesLocked` so it surfaces
  swallowed errors via `c.Logger`. Tracked separately in
  [`assetmin/docs/CHECK_PLAN.md`](../../assetmin/docs/CHECK_PLAN.md)
  follow-up notes.
- Caching/perf considerations for very large module trees. Current
  `UnobservedFiles` filters cover the common cases; add module-root
  exclusion lists if real projects show event storms.
