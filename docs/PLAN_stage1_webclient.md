# PLAN Stage 1 — Register WebClientGenerator handler

## Goal
Register the new `WebClientGenerator` handler (introduced in `tinywasm/client` Stage 3) with the TUI build section, and update the call site for the `CreateDefaultWasmFileClientIfNotExist` break change.

## Dependency
`tinywasm/client` Stage 3 must be completed and the new version published before executing this plan.

## Break Change to Fix

`WasmClient.CreateDefaultWasmFileClientIfNotExist()` now requires a `bool` parameter (`skipIDEConfig`).
The existing call in `handler_lifecycle.go:21` passes no argument — it must be updated to pass `false`
so that IDE config generation behavior remains unchanged for the automatic project-root invocation.

```go
// Before
h.WasmClient.CreateDefaultWasmFileClientIfNotExist()

// After
h.WasmClient.CreateDefaultWasmFileClientIfNotExist(false)
```

## New Handler Registration

In `section-build.go`, inside `InitBuildHandlers()`, after the existing `AddHandler` call for `h.WasmClient`,
add one line to register the generator button with the same color:

```go
h.Tui.AddHandler(h.WasmClient.WebClientGenerator(), colorPurpleMedium, h.SectionBuild)
```

The button appears in the BUILD tab immediately after the WASM build-mode edit field.
It shares the same purple color to signal it belongs to the same subsystem.

## Files to Modify

| File | What changes |
|------|-------------|
| [go.mod](../go.mod) | Bump `github.com/tinywasm/client` to the version that includes Stage 3, run `go mod tidy` |
| [handler_lifecycle.go](../handler_lifecycle.go) | Line 21: add `false` argument to `CreateDefaultWasmFileClientIfNotExist` |
| [section-build.go](../section-build.go) | After line 160 (`AddHandler(h.WasmClient, ...)`): add `h.Tui.AddHandler(h.WasmClient.WebClientGenerator(), colorPurpleMedium, h.SectionBuild)` |

## Steps

- [ ] Update `go.mod`: bump `github.com/tinywasm/client` to the version containing `WebClientGenerator` and the `skipIDEConfig` parameter. Run `go mod tidy`.

- [x] In `handler_lifecycle.go:21`: change `h.WasmClient.CreateDefaultWasmFileClientIfNotExist()` to `h.WasmClient.CreateDefaultWasmFileClientIfNotExist(false)`. Also fixed in `test/wizard_repro_test.go:106`.

- [ ] In `section-build.go`: after line 160 (`h.Tui.AddHandler(h.WasmClient, colorPurpleMedium, h.SectionBuild)`), add:
  ```go
  h.Tui.AddHandler(h.WasmClient.WebClientGenerator(), colorPurpleMedium, h.SectionBuild)
  ```

- [ ] Run `gotest` in `tinywasm/app` — all must pass.

- [ ] Manual smoke test: start the app from a subfolder that has no `web/client.go`. Press the "Generate web/client.go" button in the TUI BUILD tab. Verify:
  - `web/client.go` is created in the expected path.
  - No `.vscode/` directory is created in the subfolder.
  - A second press does nothing (guard: file already exists).
  - Compilation triggers automatically after generation.
