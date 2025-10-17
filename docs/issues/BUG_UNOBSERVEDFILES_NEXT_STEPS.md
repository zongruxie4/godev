# Test Results and Next Steps for BUG_UNOBSERVEDFILES Fix

## Test Execution Summary

**Test:** `TestDeployUnobservedFilesNotProcessedByAssetmin`
**Status:** ✅ SUCCESSFULLY DETECTED THE BUG
**Date:** October 17, 2025

### Bug Confirmed

The test revealed the exact problem:

```
Goflare UnobservedFiles: [
  "/tmp/.../deploy/edgeworker/app.wasm",      ❌ ABSOLUTE PATH (should be relative)
  "deploy/edgeworker/_worker.js"              ✅ RELATIVE PATH (correct)
]
```

---

## Root Cause Identified

### Location: `tinywasm/builderInit.go:13`

```go
func (w *TinyWasm) builderWasmInit() {
	sourceDir := path.Join(w.AppRootDir, w.Config.SourceDir)
	outputDir := path.Join(w.AppRootDir, w.Config.OutputDir)  // ❌ CREATES ABSOLUTE PATH
	mainInputFileRelativePath := path.Join(sourceDir, w.Config.MainInputFile)

	baseConfig := gobuild.Config{
		// ...
		OutFolderRelativePath: outputDir,  // ❌ PASSING ABSOLUTE PATH AS "RELATIVE"
	}
}
```

**Problem:**
- `outputDir` is created by joining `AppRootDir` (absolute) + `OutputDir` (relative)
- This creates an **absolute path** like `/tmp/test/deploy/edgeworker`
- This absolute path is then passed to `gobuild.Config.OutFolderRelativePath`
- When `FinalOutputPath()` is called, it returns this absolute path
- When `OutputRelativePath()` calls `FinalOutputPath()`, it gets an absolute path
- This causes inconsistency in `UnobservedFiles()` which expects relative paths

---

## Impact Analysis

### Affected Components

1. **tinywasm** - Creates absolute paths internally
2. **goflare** - Uses `OutputRelativePath()` expecting relative path
3. **devwatch** - Receives mixed absolute/relative paths in UnobservedFiles
4. **assetmin** - May process `_worker.js` due to path matching failure

### Current Behavior

| Method | Expected Return | Actual Return | Status |
|--------|----------------|---------------|--------|
| `OutputRelativePath()` | `deploy/edgeworker/app.wasm` | `/tmp/.../deploy/edgeworker/app.wasm` | ❌ BROKEN |
| `filepath.Join(config, "_worker.js")` | `deploy/edgeworker/_worker.js` | `deploy/edgeworker/_worker.js` | ✅ WORKS |

---

## Proposed Solutions

### Option A: Fix tinywasm to Return Relative Paths (Recommended)

**Change `tinywasm/builderInit.go:92-94`:**

```go
// returns the RELATIVE path to the final output file eg: deploy/edgeworker/app.wasm
func (w *TinyWasm) OutputRelativePath() string {
	// FinalOutputPath returns absolute path, extract relative portion
	fullPath := w.activeBuilder.FinalOutputPath()
	
	// Remove AppRootDir prefix to get relative path
	if strings.HasPrefix(fullPath, w.Config.AppRootDir) {
		relPath := strings.TrimPrefix(fullPath, w.Config.AppRootDir)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		return relPath
	}
	
	// Fallback: construct from config
	return filepath.Join(w.Config.OutputDir, w.Config.OutputName+".wasm")
}
```

**Pros:**
- ✅ Fixes the method to match its name and documentation
- ✅ Benefits all consumers of tinywasm
- ✅ Makes tinywasm API more consistent
- ✅ No changes needed in goflare

**Cons:**
- ⚠️ Requires changes to tinywasm package
- ⚠️ Need to verify no breaking changes for other consumers

---

### Option B: Fix goflare to Not Use OutputRelativePath (Alternative)

**Change `goflare/events.go`:**

```go
func (h *Goflare) UnobservedFiles() []string {
	return []string{
		// Construct paths directly from config (already relative)
		filepath.Join(h.config.RelativeOutputDirectory, h.config.OutputWasmFileName),
		filepath.Join(h.config.RelativeOutputDirectory, h.outputJsFileName),
	}
}
```

**Pros:**
- ✅ Simpler fix (only changes goflare)
- ✅ Self-contained solution
- ✅ No breaking changes to tinywasm

**Cons:**
- ⚠️ Doesn't fix underlying issue in tinywasm
- ⚠️ Duplicates path construction logic
- ⚠️ OutputRelativePath() remains misleading

---

### Option C: Hybrid Approach

1. **Fix goflare immediately** (Option B) to solve the immediate bug
2. **Fix tinywasm** (Option A) in a separate PR for long-term consistency
3. **Update goflare again** to use fixed `OutputRelativePath()`

**Pros:**
- ✅ Quick fix for immediate bug
- ✅ Long-term proper fix
- ✅ Staged rollout reduces risk

---

## Decision Required

### Questions for You

1. **Which option do you prefer?**
   - Option A: Fix tinywasm (proper fix but touches more code)
   - Option B: Fix goflare only (quick fix)
   - Option C: Hybrid approach (staged fix)

2. **Should we also add a similar fix to `WasmExecJsOutputPath()`?**
   - Currently returns absolute: `/home/.../deploy/edgeworker/wasm_exec.js`
   - Should return relative: `deploy/edgeworker/wasm_exec.js`
   - Note: This file is not currently created (DisableWasmExecJsOutput=true)

3. **Should we add more test coverage?**
   - Unit test for `OutputRelativePath()` in tinywasm
   - Integration test for path consistency across packages
   - Test for other consumers of tinywasm

---

## Immediate Next Steps (Waiting for Your Decision)

### If Option A (Fix tinywasm):
1. [ ] Modify `tinywasm/builderInit.go` - `OutputRelativePath()`
2. [ ] Optionally fix `WasmExecJsOutputPath()` for consistency
3. [ ] Add unit test in tinywasm for path return types
4. [ ] Run existing tinywasm tests to verify no breakage
5. [ ] Update goflare test expectations
6. [ ] Run golite full test suite
7. [ ] Test manually in golite/example

### If Option B (Fix goflare only):
1. [ ] Modify `goflare/events.go` - `UnobservedFiles()`
2. [ ] Update test expectations
3. [ ] Run golite test suite
4. [ ] Test manually in golite/example
5. [ ] Document limitation in tinywasm (optional)

### If Option C (Hybrid):
1. [ ] Implement Option B immediately
2. [ ] Verify fix works
3. [ ] Create separate issue/PR for Option A
4. [ ] Implement Option A later
5. [ ] Switch goflare back to using fixed method

---

## Test Verification Plan

After implementing the fix:

1. **Run unit test:**
   ```bash
   cd /home/cesar/Dev/Pkg/Mine/golite
   go test -v -run TestDeployUnobservedFilesNotProcessedByAssetmin
   ```
   Expected: Test should PASS with both paths relative

2. **Manual test:**
   ```bash
   cd /home/cesar/Dev/Pkg/Mine/golite/example
   golite
   # Check logs - should NOT see: "ASSETS .js create ... deploy/edgeworker/_worker.js"
   ```

3. **Verify main.js:**
   ```bash
   cat src/web/public/main.js | grep -i "worker\|fetch"
   ```
   Expected: Should NOT contain _worker.js content

---

## Recommendation

**My Recommendation: Option B (Quick Fix) followed by Option A (Proper Fix)**

**Reasoning:**
1. Option B solves the immediate bug with minimal risk
2. goflare already has the config values it needs
3. tinywasm fix can be done properly with more testing
4. Staged approach reduces risk of breaking other consumers

**Implementation Order:**
1. Fix goflare now (5 minutes)
2. Test and verify (10 minutes)
3. Create issue for tinywasm fix (5 minutes)
4. Implement tinywasm fix when ready (30 minutes)
5. Update goflare to use fixed method (5 minutes)

---

## Awaiting Your Decision

Please indicate:
- [ ] Option A - Fix tinywasm
- [ ] Option B - Fix goflare only  
- [ ] Option C - Hybrid approach
- [ ] Other approach

Once you decide, I'll proceed with the implementation.
