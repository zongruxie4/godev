# PLAN — Hot-reload auto-discovery for local transitive packages

> **Status:** deferred. Phase 2 follow-up to
> [`CHECK_PLAN.md`](CHECK_PLAN.md) (the module-root fix). Open this when
> the module-root patch is merged and verified, and a real user reports
> the N-packages-in-other-dirs case.

## Why this exists

The current fix in `CHECK_PLAN.md` registers the **module root** with
devwatch, so sibling subpackages of the active project (within the same
Go module) get hot-reload. It also keeps the existing behaviour of
watching every dir in `replace` directives.

That covers two cases:

1. Active project is `foo/web/`, edited file is `foo/sub/ssr.go`
   (same module). ✅ Phase 1.
2. Active project depends on `bar`, declared in `go.mod` as
   `replace github.com/.../bar => ../../bar`. ✅ already supported.

It does **not** cover:

3. Active project depends on `bar` that lives locally on disk but is
   **not** declared via `replace` (e.g. resolved via `GOPATH`,
   `go.work`, vendor, or a local `pkg/mod` cache the dev is editing).
   The dev edits `bar/baz/ssr.go` and gets no hot-reload.

This case matters because the **real intent of `tinywasm/app` is "dev
tool that hot-reloads anything Go-relevant on disk that the project
actually uses"**. Forcing the dev to add a `replace` directive every
time they touch a sibling repo is friction the orchestrator should
absorb.

## Proposed approach

Auto-discover all packages the project's build actually imports, filter
to the ones whose source is on local disk (not in
`$GOPATH/pkg/mod/cache`), and register their directories with the
watcher.

### Sketch

```go
// In section-build.go, after the existing module-root + replace wiring:

cmd := exec.Command("go", "list", "-deps", "-json", "./...")
cmd.Dir = h.Config.RootDir
out, err := cmd.Output()
if err != nil {
    h.Watcher.Logger("WATCH", "auto-discovery skipped:", err)
} else {
    dirs := parseLocalPackageDirs(out) // filter Standard==false, Dir not under pkg/mod cache
    if len(dirs) > 0 {
        h.Watcher.Logger("WATCH", "auto-discovered local deps:", len(dirs))
        h.Watcher.AddDirectoriesToWatch(dirs...)
    }
}
```

`parseLocalPackageDirs` filters each `go list` package object:
- Skip if `Standard == true` (stdlib).
- Skip if `Dir` starts with `$GOPATH/pkg/mod/` or `$GOMODCACHE/`
  (immutable cached deps).
- Keep otherwise. These are local, editable sources.

## Trade-offs

| Concern | Phase 1 (module root) | Phase 2 (auto-discovery) |
|---|---|---|
| LOC | ~5 | ~50 + tests |
| Dev configures `replace`? | Yes for cross-module | No |
| Startup cost | None | `go list -deps` ≈ 200–500 ms |
| Re-discovery on import changes | N/A | Need to re-run when imports change (could hook into the existing `go.mod` change handler — first cut: only at startup) |
| Risk of watching irrelevant dirs | Low | Medium — must filter pkg/mod aggressively |
| Failure mode | Silent miss for case (3) | If `go list` fails the cache, dev gets degraded UX with a logged warning |

## Open questions

1. **Re-discovery cadence.** First implementation: only at startup. Is
   that enough? `go.mod` changes already trigger a rescan path
   (`GoModHandler.NewFileEvent` reconciles replace paths) — could
   piggyback on that to re-run discovery.
2. **`go.work` workspaces.** `go list -deps` already respects
   `go.work` if present. Confirm with a workspace fixture in tests.
3. **Vendor directory.** If `vendor/` exists, `go list` may resolve
   deps through it. Should `vendor/` be watched? Probably no — it is a
   build artifact, not source. Add to `UnobservedFiles` if not already.
4. **Performance ceiling.** What does `go list -deps` cost on a real
   monorepo with 500+ packages? Need to measure before assuming
   acceptable.
5. **Interaction with `ListModulesFn`.** Tests already use
   `h.ListModulesFn` to inject module sets. Auto-discovery should
   respect that override, not bypass it.

## Test plan

- `app/test/hotreload_autodiscover_test.go`:
  - Fixture: `parent/go.mod` (project), `external/go.mod` (separate
    module on disk, no `replace` in parent — just imported via
    `go.work` or `GOPATH`-style override the test sets up).
  - Edit `external/somepkg/ssr.go`.
  - Assert the watcher's directory set includes `external/somepkg/`
    after `InitBuildHandlers`.
  - The reflection helper from `hotreload_subpackage_test.go` is reused.

## Files affected

- `app/section-build.go` — add discovery call + helper.
- `app/section-build.go` (or new `app/local_deps.go`) — `parseLocalPackageDirs`.
- `app/test/hotreload_autodiscover_test.go` — regression test.

## Out of scope

- Watching arbitrary directories the user lists in a config file.
  Convention over configuration: if `go list` says it's used, watch
  it. Period.
- Replacing the existing `replace`-based wiring. Auto-discovery is
  additive, not a replacement — `replace` continues to work
  identically and remains the explicit signal for "I'm developing
  this module".
- A full filesystem indexer / glob-based watcher. The `go list` path
  is sufficient and stays aligned with the Go toolchain's view of
  "what this project actually uses".

## Decision deferred

Phase 1 (`CHECK_PLAN.md`) ships first because:
- It fixes the demonstrable bug in the current repro (`platformd`
  sibling subpackage).
- It is small, safe, and lands without architectural debate.
- It does not preclude Phase 2 — the calls are additive.

Phase 2 ships when:
- Phase 1 has been in use long enough to confirm no regression in the
  watch-set widening.
- A real consumer reports needing case (3) above (cross-module local
  edits without `replace`).
- Performance of `go list -deps` is measured on a real monorepo.

If neither of those happens within a quarter, this plan can be closed
as "convention-over-configuration is enough".
