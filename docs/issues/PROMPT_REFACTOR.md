# Refactoring Plan for TinyWasm

## Goal Description
Wire `tinywasm` and `assetmin` together in `tinywasm` using the new shared `Store` and direct update mechanisms.

## Proposed Changes

### TinyWasm

#### [MODIFY] [section-build.go](file:///home/cesar/Dev/Pkg/Mine/tinywasm/section-build.go) (or relevant handler file)
- **Implement Store**: Ensure `tinywasm`'s handler implements the `Store` interface required by `tinywasm`.
    - `Get(key string) (string, error)`
    - `Set(key, value string) error`
    - It likely already has a DB instance; we might need to wrap it or ensure it satisfies the interface.
- **Configure TinyWasm**:
    - Inject the `Store` into `tinywasm.Config`.
    - Set `DisableWasmExecJsOutput = true` in `tinywasm.Config` to prevent it from writing the file itself.
- **Wire Notifications**:
    - Set `tinywasm.Config.OnWasmExecChange` callback.
    - Inside the callback:
        1.  Get the new `wasm_exec.js` content from `tinywasm`.
        2.  Call `assetmin.UpdateAssetContent("wasm_exec.js", content)` to update it directly in `assetmin`.
- **Update Build Section**:
    - Ensure `AddSectionBUILD` sets up this wiring correctly during initialization.

## Verification Plan
### Manual Verification
- Verify that changing compilation mode in `tinywasm` (via UI/shortcuts) updates the `wasm_exec.js` served/minified by `assetmin` without creating intermediate files.
- Verify that the compilation mode is persisted across restarts (via Store).
