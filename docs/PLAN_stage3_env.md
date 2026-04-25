# PLAN Stage 3 — .env always at project root (go.mod level)

## Goal
The `.env` file (kvdb database) must always be created at the root of the Go project
(where `go.mod` lives), never in a subdirectory.
If no initialized Go project exists, tinywasm must exit with a clear error — it cannot
and must not create `.env` or any other file in an arbitrary directory.

## Problem

### Bug A — `.env` created in wrong directory
`cmd/tinywasm/main.go:56`:
```go
db, err := kvdb.New(filepath.Join(startDir, ".env"), ...)
```
`startDir` is the raw `os.Getwd()` result. When `FindProjectRoot` succeeds (line 43),
`projectRoot` is set correctly — but line 56 still uses `startDir`.
Result: running `tinywasm` from a subdirectory creates `.env` in that subdirectory.

### Bug B — `.env` created before project validation
`kvdb.New` runs before `Bootstrap`/`Start` validates whether a Go project exists.
A new `.env` is created even when the directory has no `go.mod`.

Both bugs pollute the filesystem and make `.env` unreliable as a project config store.

## Dependency
Stage 2 must be completed first. Both stages modify `cmd/tinywasm/main.go` — applying them
out of order causes merge conflicts. There is no logical dependency between them.

---

## Change 1 — `main.go`: use `projectRoot` for kvdb path

One-line fix. `projectRoot` is already available from line 43.

### Before (line 56)
```go
db, err := kvdb.New(filepath.Join(startDir, ".env"), logger.Logger, &app.FileStore{})
```

### After
```go
db, err := kvdb.New(filepath.Join(projectRoot, ".env"), logger.Logger, &app.FileStore{})
```

`projectRoot` must be declared in the outer scope (moved out of the `if` block at line 43).

### Files
| File | Change |
|------|--------|
| [cmd/tinywasm/main.go](../cmd/tinywasm/main.go) | Declare `projectRoot` before the `if` block; use it for kvdb path |

---

## Change 2 — Guard: exit if no project found AND directory is not empty

When `FindProjectRoot` fails, there are two valid situations:
- **Empty directory**: the wizard should still run to initialize a new project. Allow Bootstrap to proceed — kvdb will not be created (no `Set` call happens before go.mod exists).
- **Non-empty directory without go.mod**: this is user error. Exit with a clear message.

```go
projectRoot, err := devflow.FindProjectRoot(startDir)
if err != nil {
    // Allow empty directories through — the wizard will initialize them.
    // Non-empty dirs without go.mod are an error: tinywasm cannot work there.
    entries, _ := os.ReadDir(startDir)
    hasFiles := false
    for _, e := range entries {
        n := e.Name()
        if n != ".git" && n != ".DS_Store" {
            hasFiles = true
            break
        }
    }
    if hasFiles {
        twfmt.Print("Directory", "Not", "Initialized")
        os.Exit(1)
    }
    // Empty dir: set projectRoot = startDir so kvdb path is consistent if wizard creates go.mod
    projectRoot = startDir
}
```

**Message composition**: use only words already present in `tinywasm/fmt/dictionary`:
- "Directory" → "Directorio" ✓ (in dictionary)
- "Not" → "No" ✓ (in dictionary)
- "Initialized" → "Inicializado" ✓ (in dictionary)

Do NOT add a `messages/` subpackage. Do NOT register "Go" (it is the same in all languages),
"Not" or "Initialized" — they already exist in `tinywasm/fmt/dictionary`.
Do NOT import `"fmt"` (stdlib) — `twfmt` provides all translation and print utilities.

**IMPORTANT — `messages/` subpackage**: Do NOT create `messages/messages.go`.
All required words are already registered by `_ "github.com/tinywasm/fmt/dictionary"`.
Adding a subpackage that re-registers existing words or trivially maps a word to itself
("Go"→"Go") is noise and must be avoided.

### Files
| File | Change |
|------|--------|
| [cmd/tinywasm/main.go](../cmd/tinywasm/main.go) | Guard before `kvdb.New` (with empty-dir exception); import `twfmt "github.com/tinywasm/fmt"` + `_ "github.com/tinywasm/fmt/dictionary"` |

---

## Change 3 — `Bootstrap`: secondary guard in `Start` for subdirectory

`start.go` already rejects home dir and `/`. Add rejection when `startDir` is inside
a Go project but is not its root (i.e. the user ran from a subdirectory):

```go
// After the existing home/root check:
if root, err := devflow.FindProjectRoot(startDir); err == nil && root != startDir {
    logger(twfmt.Translate("Directory", "Not", "Initialized").String())
    return false
}
```

Note: do NOT include "Go" in the message — it is the same word in all languages and adds no
information. "Directory Not Initialized" → "Directorio No Inicializado" is clear and correct.

Use the package-level `devflow.FindProjectRoot` (not a method — `GoHandler` does not expose it).
`devflow` is already imported in `start.go`.

This is a safety net for call sites other than `main.go` (e.g. tests, daemon sub-project launch).
The main guard in `main.go` (Change 2) handles the normal CLI path.

### Files
| File | Change |
|------|--------|
| [start.go](../start.go) | Add subdirectory guard after existing home/root check (lines 58-62) |

---

## Tests

### Test 1a — `TestEnv_BugReproduction_StartDir` (env_test.go) — DOCUMENTS BUG ✓ created
Calls `kvdb.New` with `startDir` (the bug). Asserts `.env` lands in subdirectory.
Always passes — exists to make the bug explicit and detectable if behavior changes.
If this test starts failing, the bug is gone and the test must be removed.

### Test 1b — `TestEnv_CreatedAtProjectRoot_NotSubdirectory` (env_test.go) — REGRESSION GUARD ✓ created
Calls `kvdb.New` with `projectRoot` (the fix). Asserts `.env` at root, NOT in sub/.
This is the contract: if someone regresses main.go back to startDir, this fails.

Both tests are already written in [env_test.go](../env_test.go) and passing.

### Test 2 — No `.env` created when no go.mod exists (env_test.go)
```
Setup:
  tmpDir/ ← no go.mod

Call FindProjectRoot(tmpDir) → must return error.
Assert: no .env file anywhere in tmpDir.
```

### Test 3 — `Start` returns false when startDir is inside a project but not its root (start_test.go)
```
Setup:
  root/go.mod
  root/sub/  ← startDir

Call Start(root/sub, ...) → must return false.
Assert return value is false only. No message check, no file check (non-fragile).

Write this test BEFORE applying Changes 2 and 3 — it must fail first to confirm
the test is actually exercising the new guard code.
```

---

## Rules for `cmd/tinywasm/main.go`

### Rule: `main()` must be a thin entry point
`main()` must only parse flags and call a single setup function. All initialization logic
(finding project root, building kvdb, constructing `BootstrapConfig`) must live in a separate
exported or unexported function (e.g. `run(cfg runConfig) error`) so it can be unit-tested
without spawning a process.

**Good:**
```go
func main() {
    debugFlag := flag.Bool("debug", false, "Enable debug mode")
    mcpFlag := flag.Bool("mcp", false, "Run as MCP Daemon")
    flag.Parse()
    if err := run(*debugFlag, *mcpFlag); err != nil {
        os.Exit(1)
    }
}
```

**Bad:** putting 80+ lines of initialization directly in `main()`.

### Rule: no stdlib `"fmt"` import
`twfmt "github.com/tinywasm/fmt"` already handles translation and output.
Do NOT import `"fmt"` from the stdlib alongside `twfmt`. Use `twfmt.Print(...)` or
write to os.Stdout directly if a simple newline-terminated print is needed.

### Rule: no `messages/` subpackage
Do not create `messages/messages.go` or any other subpackage that only registers
dictionary words already present in `tinywasm/fmt/dictionary`.
Only create a `messages/` package if it registers words that are genuinely missing from
the global dictionary AND are used in more than one place. Check the dictionary first.

---

## Test location

All new tests must go in `test/` (e.g. `test/env_test.go`, `test/start_test.go`).
Do NOT place `_test.go` files in the package root unless they are already there and
integrated with the existing suite. The `test/` directory is the established convention
for this project — the existing `env_test.go` and `daemon_test.go` in the root are
legacy; do not add more.

---

## Steps

- [ ] Verify tinywasm/fmt PLAN is complete (dictionary words available)
- [x] Write test 1a (bug reproduction) and test 1b (regression guard) — `env_test.go`
- [x] Write test 2 (no project → no .env) — `env_test.go`
- [ ] Write test 3 (subdirectory → `Start` returns false) — `test/start_test.go`; write BEFORE applying the fix; it must fail first
- [ ] `cmd/tinywasm/main.go`: extract logic to `run()` function; declare `projectRoot` in outer scope; add guard (with empty-dir exception); use `projectRoot` for kvdb path; import only `twfmt` (no stdlib `"fmt"`); no `messages/` import
- [ ] `start.go`: add subdirectory guard using `devflow.FindProjectRoot`
- [ ] Run `gotest` — test 3 must now pass; tests 1a, 1b, 2 must still pass
- [ ] Smoke test: run `tinywasm` from a non-empty subdirectory without go.mod → clean exit with translated message, no `.env` created
- [ ] Smoke test: run `tinywasm` from an empty directory → wizard opens normally
