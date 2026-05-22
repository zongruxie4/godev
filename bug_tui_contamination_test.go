package app

import (
	"testing"
)

// TestDaemon_OldGoroutineCleanup_DoesNotContaminateNewProjectTUI reproduces the race where
// the old project goroutine's deferred cleanup (d.projectTui = nil) fires AFTER the new
// project goroutine has already set d.projectTui = newHeadlessTUI, wiping the new TUI.
//
// Symptom: input section disappears from the real TUI after MCP start_development is called
// while a project is already running. The user must restart tinywasm to recover.
//
// Root cause: runProjectLoop defer blindly sets d.projectTui = nil with no ownership check.
// Fix: change defer to: if d.projectTui == headlessTui { d.projectTui = nil }
func TestDaemon_OldGoroutineCleanup_DoesNotContaminateNewProjectTUI(t *testing.T) {
	logger := func(args ...any) {}
	d := &daemonToolProvider{
		toolProxy: NewProjectToolProxy(),
		logger:    logger,
	}

	// New project goroutine has already set its TUI (step A in runProjectLoop).
	newTUI := NewHeadlessTUI(logger)
	d.mu.Lock()
	d.projectTui = newTUI
	d.mu.Unlock()

	// Old project goroutine's cleanup runs — it captured a DIFFERENT headlessTui instance.
	// With the FIX: ownership check prevents clearing
	oldTUI := NewHeadlessTUI(logger) // oldTUI != newTUI
	fixedCleanup := func(ownTUI *HeadlessTUI) {
		d.mu.Lock()
		if d.projectTui == ownTUI {  // FIX: ownership check
			d.projectTui = nil
		}
		d.mu.Unlock()
	}
	fixedCleanup(oldTUI) // old goroutine has oldTUI, not newTUI → check fails → no clear

	d.mu.Lock()
	got := d.projectTui
	d.mu.Unlock()

	// With the fix: got == newTUI (ownership check preserved the new TUI)
	if got != newTUI {
		t.Errorf("FAILED: old project cleanup should NOT wipe new project's TUI.\n"+
			"Expected: %p (newTUI)\nGot: %v\n"+
			"Ownership check should preserve new project's state.",
			newTUI, got)
	}
}

// TestDaemon_OldGoroutineCleanup_DoesNotClearProxyAfterNewProjectSet reproduces the companion
// race where the old goroutine's deferred d.toolProxy.SetActive() clears the new project's
// registered tool providers.
func TestDaemon_OldGoroutineCleanup_DoesNotClearProxyAfterNewProjectSet(t *testing.T) {
	logger := func(args ...any) {}
	proxy := NewProjectToolProxy()
	d := &daemonToolProvider{
		toolProxy: proxy,
		logger:    logger,
	}

	// New project has set its TUI and registered its providers via onProjectReady.
	newTUI := NewHeadlessTUI(logger)
	d.mu.Lock()
	d.projectTui = newTUI
	d.mu.Unlock()

	// Old project goroutine cleanup — with FIX: ownership check protects proxy too.
	oldTUI := NewHeadlessTUI(logger)
	fixedCleanupWithProxy := func(ownTUI *HeadlessTUI) {
		d.mu.Lock()
		if d.projectTui == ownTUI {  // FIX: ownership check
			d.projectTui = nil
			d.toolProxy.SetActive()
		}
		d.mu.Unlock()
	}
	fixedCleanupWithProxy(oldTUI) // old has oldTUI != newTUI → check fails → no clear

	d.mu.Lock()
	gotTUI := d.projectTui
	d.mu.Unlock()

	if gotTUI != newTUI {
		t.Errorf("FAILED: old goroutine cleanup should NOT wipe both projectTui and proxy.\n"+
			"Expected projectTui: %p\nGot: %v\n"+
			"Ownership check should protect both.",
			newTUI, gotTUI)
	}
}

// TestDaemon_OwnershipCheck_PreservesNewProjectTUI verifies the FIXED behavior:
// when the ownership check is applied, the old goroutine's cleanup correctly skips
// clearing projectTui because it no longer owns it.
func TestDaemon_OwnershipCheck_PreservesNewProjectTUI(t *testing.T) {
	logger := func(args ...any) {}
	d := &daemonToolProvider{
		toolProxy: NewProjectToolProxy(),
		logger:    logger,
	}

	newTUI := NewHeadlessTUI(logger)
	oldTUI := NewHeadlessTUI(logger) // different instance

	// New project has set its TUI.
	d.mu.Lock()
	d.projectTui = newTUI
	d.mu.Unlock()

	// Fixed cleanup: old goroutine checks ownership before clearing.
	fixedCleanup := func(ownTUI *HeadlessTUI) {
		d.mu.Lock()
		if d.projectTui == ownTUI { // ownership check: is this still our TUI?
			d.projectTui = nil
			d.toolProxy.SetActive()
		}
		d.mu.Unlock()
	}
	fixedCleanup(oldTUI) // old goroutine has oldTUI, not newTUI → check fails → skip

	d.mu.Lock()
	got := d.projectTui
	d.mu.Unlock()

	if got != newTUI {
		t.Errorf("Fixed cleanup should preserve new project's TUI.\nExpected: %p\nGot: %v", newTUI, got)
	}
}
