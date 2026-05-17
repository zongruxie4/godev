package app

// Reproducer test suite for docs/PLAN.md (external server mode orchestration fix).
//
// Every test is skipped today because the new APIs do not yet exist:
//   - tinywasm/assetmin: EnableSSRMode(), FlushToDisk() error
//   - tinywasm/server:   SetBeforeExternalServerStart(func() error)
//   - tinywasm/client:   UseDiskStorage(), UseMemoryStorage()  (replaces SetBuildOnDisk)
//
// The external agent implementing PLAN.md MUST:
//   1. Remove every t.Skip in this file.
//   2. Wire the calls per docs/PLAN.md §2.
//   3. Make every test pass.

import (
	"testing"
)

// B2/B3 — Every in-memory asset must be on disk BEFORE strategy.Start runs.
func TestExternalMode_FlushesAllAssetsBeforeStart(t *testing.T) {
	t.Skip("see docs/PLAN.md — requires BeforeExternalServerStart hook + FlushToDisk")

	// TODO(agent):
	// 1. Build a Handler with a real assetmin.AssetMin and a fake server whose
	//    strategy records the moment Start() is invoked.
	// 2. Register N>5 in-memory assets via NewFileEvent.
	// 3. Place a web/server.go stub so the server detects external mode.
	// 4. Call h.Server.StartServer(&wg).
	// 5. Assert: for every registered asset, the file in h.Config.WebPublicDir()
	//    exists with the in-memory minified bytes at the timestamp recorded just
	//    before strategy.Start.
}

// B2 — Strict synchronous order: client → assetmin → strategy.Start.
func TestExternalMode_StartOrderIsSynchronous(t *testing.T) {
	t.Skip("see docs/PLAN.md — requires synchronous BeforeExternalServerStart hook")

	// TODO(agent):
	// Record event names ("client.UseDiskStorage", "client.Compile",
	// "assetmin.FlushToDisk", "strategy.Start") via callbacks/fakes; assert the
	// slice matches that order.
}

// New — Flush errors must abort the transition; strategy.Start must NOT run.
func TestExternalMode_FlushErrorAbortsServerStart(t *testing.T) {
	t.Skip("see docs/PLAN.md — requires error propagation from FlushToDisk")

	// TODO(agent):
	// Inject an assetmin (or its OutputDir) that forces FlushToDisk to fail.
	// Assert strategy.Start is NEVER called and an error is logged.
}

// B3 — InitBuildHandlers must enable SSR mode via the new explicit API,
// without registering any Go compiler.
func TestExternalMode_InitEnablesSSRWithoutCompiler(t *testing.T) {
	t.Skip("see docs/PLAN.md — requires EnableSSRMode() in section-build.go init")

	// TODO(agent):
	// After h.InitBuildHandlers(), assert assetmin is in SSR mode (CSS hot-reload
	// branch active) AND no SSR compiler is registered (SetSSRCompiler never called).
}

// §1 — The hook fires on EVERY external-mode StartServer, not only the first.
func TestExternalMode_HookFiresOnEveryExternalStart(t *testing.T) {
	t.Skip("see docs/PLAN.md — requires hook idempotency contract")

	// TODO(agent):
	// 1. Place web/server.go stub so external mode is sticky.
	// 2. Call StartServer twice (with a Stop in between).
	// 3. Assert the hook was invoked exactly twice (counted via a closure variable).
}

// §1 — RestartServer must NOT invoke the hook (by design).
func TestExternalMode_RestartDoesNotFireHook(t *testing.T) {
	t.Skip("see docs/PLAN.md — requires RestartServer to bypass the hook")

	// TODO(agent):
	// 1. Boot in external mode (hook fires once during StartServer).
	// 2. Call h.Server.RestartServer().
	// 3. Assert the hook counter did NOT increment.
}
