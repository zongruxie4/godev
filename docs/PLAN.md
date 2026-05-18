# PLAN: Fix `-dev` Flag Contract with External Server

## Bug Report

`app/section-build.go` passes `-dev` to the compiled external server binary when `h.DevMode == true`. The auto-generated server template (`server/templates/server_basic.md`) does not define this flag, so any project using the default generated `web/server.go` crashes in dev mode:

```
flag provided but not defined: -dev
exit status 2
```

## Root Cause

**`app` makes an undocumented assumption** that every external server it manages will accept a `-dev` flag. This is an implicit contract that is not enforced anywhere.

Relevant code — `app/section-build.go:115-125`:

```go
srv.SetRunArgs(func() []string {
    args := []string{
        "-public-dir=" + filepath.Join(h.RootDir, h.Config.WebPublicDir()),
        "-port=" + h.Config.ServerPort(),
    }
    if h.DevMode {
        args = append(args, "-dev")  // <-- passed but not defined in template
    }
    return append(args, h.WasmClient.ArgumentsForServer()...)
})
```

## Options

### Option A — Fix the template (preferred, in `tinywasm/server`)
Add `-dev` flag to the server template so the generated server accepts it.
See `server/docs/PLAN.md` for details.

**Pros:** Contract is fulfilled at the source. No changes needed in `app`.  
**Cons:** Already-generated files need manual patch.

### Option B — Make `-dev` optional from `app` side
Instead of always passing `-dev`, probe whether the binary accepts it (e.g. run `./server -help` and check output before starting).

**Pros:** Backward compatible with existing user servers that don't define `-dev`.  
**Cons:** Complex; adds startup latency; fragile.

### Option C — Document the contract
Add a doc comment in `SetRunArgs` and in `README.md` stating that any external server **must** accept `-dev`, `-port`, and `-public-dir` flags.

**Pros:** Zero code change.  
**Cons:** Doesn't fix current broken projects.

## Recommended Fix

**Option A + Option C.**

1. Fix `server/templates/server_basic.md` (tracked in `server/docs/PLAN.md`).
2. Add comment in `section-build.go` near `SetRunArgs` documenting the required flags contract.
3. Update `app/README.md` section "Backend (`web/server.go`)" to list required flags.

## Files to Change

- `app/section-build.go` — add contract comment near `-dev` append
- `app/README.md` — document required external server flags

## Affected Libraries

- `github.com/tinywasm/app` — owns the caller (document contract here)
- `github.com/tinywasm/server` — owns the template (fix in `server/docs/PLAN.md`)
