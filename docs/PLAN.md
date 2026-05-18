# PLAN: Simplify Server Run Args — Only Pass `-server_port`

## Problem

`app/section-build.go` builds the run args for the external server and appends:
- `-public-dir=...`
- `-port=...`
- `-dev` (when DevMode)
- everything from `h.WasmClient.ArgumentsForServer()` (e.g. `-wasmsize_mode`)

The server template uses Go's `flag` package, which crashes on unknown flags.
Any arg `app` or `client` adds silently breaks all existing generated servers.

## Solution — Pass only `-server_port`, drop the rest

After the server template is simplified (see `server/docs/PLAN.md`):

- `-server_port` replaces `-port` (new agreed name)
- `-public-dir` is dropped — server always serves from its working dir (`web/public`)
- `-dev` is dropped — server doesn't need to know about dev mode
- `h.WasmClient.ArgumentsForServer()` is **not passed to the server** — wasm size mode
  is a compiler concern, not a server concern

### New `SetRunArgs` in `section-build.go`

```go
srv.SetRunArgs(func() []string {
    return []string{"-server_port=" + h.Config.ServerPort()}
})
```

That's it. One arg. The server ignores everything it doesn't recognize.

## Files to Change

| File | Change |
|------|--------|
| `app/section-build.go` | Simplify `SetRunArgs` to only `-server_port` |

## Dependency

Requires `server` to be published first with the new `lookupArg`-based template
(see `server/docs/PLAN.md`).

## Stage Checklist

- [ ] `server` published with simplified template
- [ ] Update `SetRunArgs` in `section-build.go`
- [ ] Run `gotest` in `tinywasm/app`
- [ ] Publish with `gopush`
