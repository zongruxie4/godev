# PLAN — Fix: Hot Reload for Nested SSR Sub-packages

## Problem

When a project places `ssr.go` inside a multi-level sub-package (e.g.
`modules/contact/`) rather than at the module root, two bugs in
`tinywasm/assetmin` prevent the CSS from ever being loaded:

1. **Initial scan**: `moduleSubpackagesUsed` drops sub-paths that contain `"/"`,
   so `modules/contact` is never discovered by `LoadSSRModules`.

2. **Hot reload**: `ExtractSSRAssets` requires `go.mod` to be present in the
   exact directory passed to it. Sub-packages share the root `go.mod`, so
   `ReloadSSRModule(contactDir)` always returns "no go.mod found".

Observed in production: `goflare-demo/modules/contact/ssr.go` CSS is never
applied; browser shows unstyled form after every edit.

Log evidence:
```
Initial SSR load error: ssr.go not found in /home/cesar/Dev/Project/tinywasm/goflare-demo
```

Failing tests (added as part of this investigation):
- `assetmin/tests/TestBug_DeepSubpackage_NotLoadedOnInitialScan`
- `assetmin/tests/TestBug_DeepSubpackage_HotReloadFails`
- `app/TestSSRHotReload_DeepSubpackage`

---

## Root cause location

All fixes are in **`tinywasm/assetmin`** — see
[`assetmin/docs/PLAN.md`](../../assetmin/docs/PLAN.md) for the detailed fix plan.

The `app` orchestrator (`section-build.go`) is correct: it passes the right
directories; the failures are in how assetmin processes them.

---

## Execution stages

| # | Task | Repo | Done |
|---|------|------|------|
| 1 | Fix `moduleSubpackagesUsed` — allow multi-segment sub-paths | `assetmin` | [ ] |
| 2 | Fix `ExtractSSRAssets` — traverse up to find go.mod before stat | `assetmin` | [ ] |
| 3 | All assetmin bug-repro tests pass | `assetmin` | [ ] |
| 4 | `app/TestSSRHotReload_DeepSubpackage` passes | `app` | [ ] |
| 5 | Verify live: edit `goflare-demo/modules/contact/ssr.go`, confirm CSS hot-reloads in browser via MCP screenshot | — | [ ] |
