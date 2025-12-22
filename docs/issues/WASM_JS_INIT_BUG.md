# WASM JavaScript Initialization Bug

## Problem Summary
AssetMin is not calling the configured `GetRuntimeInitializerJS` function when processing JavaScript files, resulting in `main.js` files that lack the proper WASM initialization code from TinyWasm.

## Current Investigation Status
✅ **STEP 1 - ASSETMIN VERIFIED**: AssetMin correctly calls `GetRuntimeInitializerJS` and includes the returned JavaScript in `main.js`. Test with mock TinyWasm handler confirms AssetMin's JavaScript processing pipeline works correctly.

✅ **STEP 2 - TINYWASM BUGS IDENTIFIED & FIXED**: Multiple critical bugs found and resolved in TinyWasm's `JavascriptForInitializing()` method.

**Critical Bugs Fixed:**
1. ✅ **Incorrect path construction**: Fixed `GetWasmExecJsPathTinyGo()` and `GetWasmExecJsPathGo()` to use correct installation patterns
   - Go: Now searches `/usr/local/go/lib/wasm/wasm_exec.js` and `/usr/local/go/misc/wasm/wasm_exec.js`
   - TinyGo: Now searches `/usr/local/lib/tinygo/targets/wasm_exec.js` and other standard patterns
2. ✅ **activeBuilder null check**: Added safety check in `JavascriptForInitializing()` to prevent nil pointer access
3. ✅ **Cross-platform compatibility**: Implemented multiple search patterns for different installation methods

**Test Results (After Fixes):**
- ✅ Coding mode: 17142 bytes JS, 1598360 bytes WASM (Go compiler)
- ✅ Debugging mode: 16631 bytes JS, 109803 bytes WASM (TinyGo -opt=1)
- ✅ Production mode: 16631 bytes JS, 14141 bytes WASM (TinyGo -opt=z)
- ✅ JavaScript generation works correctly across all compiler modes
- ✅ Cache clearing functionality working properly
- ✅ File compilation working with appropriate size differences
- ✅ **IMPORTANT**: Go vs TinyGo generate different JavaScript (17142 vs 16631 bytes), but debugging and production modes use the same TinyGo wasm_exec.js file (this is correct behavior)

**Integration Test Status:**
- ✅ AssetMin correctly calls `GetRuntimeInitializerJS` and regenerates main.js
- ✅ Mode switching triggers JavaScript regeneration
- 🔄 Minor adjustment needed: debugging vs production comparison should expect same JavaScript (both use TinyGo)

**Next Steps:**
- Test integration with AssetMin to verify end-to-end functionality
- Verify main.js generation with real file events

## Affected Components

### 1. AssetMin Package (`/assetmin/`)
- **File**: `events.go`
- **Method**: `(*AssetMin).NewFileEvent(fileName, extension, filePath, event string) error`
- **Issue**: When processing `.js` files, should trigger JavaScript regeneration
- **Method**: `(*AssetMin).startCodeJS() (string, error)`
- **Issue**: Calls `c.GetRuntimeInitializerJS()` but may not be called when TinyWasm mode changes

### 2. AssetMin Configuration (`/assetmin/`)
- **File**: `assetmin.go`
- **Config Field**: `GetRuntimeInitializerJS func() (string, error)`
- **Issue**: Function is configured but not called when JavaScript files change

### 3. TinyWasm Package (`/tinywasm/`)
- **File**: `javascripts.go`
- **Method**: `(*TinyWasm).JavascriptForInitializing() (string, error)`
- **Issue**: Generates different JavaScript for different compiler modes but AssetMin doesn't call it on mode changes

### 4. GoLDev Integration (`/tinywasm/`)
- **File**: `section-build.go`
- **Method**: `(*handler).AddSectionBUILD()`
- **Configuration**: `GetRuntimeInitializerJS: h.wasmClient.JavascriptForInitializing`

## Technical Flow

1. **Setup**: GoLDev configures AssetMin with `GetRuntimeInitializerJS` pointing to TinyWasm's method
2. **File Event**: JavaScript file changes trigger `AssetMin.NewFileEvent()`
3. **Expected**: AssetMin should call `startCodeJS()` which calls `GetRuntimeInitializerJS()` to get fresh WASM initialization code
4. **Actual**: AssetMin may not be calling the configured function or not regenerating when TinyWasm mode changes

## Investigation Plan

### Phase 1: AssetMin Verification (Current)
- Create mock TinyWasm handler that returns different JavaScript for different "modes"
- Configure AssetMin with mock handler via `setupTestEnv()`
- Verify `main.js` contains the mock WASM initialization JavaScript
- Change mock mode and verify `main.js` updates accordingly

### Phase 2: TinyWasm Investigation (Next)
- Test TinyWasm's `JavascriptForInitializing()` method directly
- Verify it returns different JavaScript for Go vs TinyGo modes
- Check if file paths for `wasm_exec.js` are correct for both compilers

## Root Cause
AssetMin's JavaScript processing pipeline may not properly invoke the configured `GetRuntimeInitializerJS` function when JavaScript files are modified, or the regeneration is not triggered when TinyWasm's compiler mode changes.

## Impact
- `main.js` files lack proper WASM initialization code
- WASM applications fail to load correctly
- Development workflow broken when using WASM projects
