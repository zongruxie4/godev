package test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"unsafe"

	"github.com/tinywasm/devwatch"
)

// TestWatcher_IncludesModuleRoot_NotJustStartDir reproduces the
// hot-reload coverage bug documented in app/docs/PLAN.md.
//
// Scenario: tinywasm is started on a wasm subpackage (e.g.
// `layout/platformd/web/`) whose Go module root sits several directories
// up (`layout/go.mod`). Sibling subpackages of that module
// (e.g. `layout/platformd/`) contain `ssr.go` files whose changes must
// trigger SSR re-extraction.
//
// Today, section-build.go only registers the start dir (and `replace`
// paths) with devwatch, so events in `layout/platformd/ssr.go` are
// never delivered to GoModHandler.NewFileEvent and the SSR cache is
// never invalidated. This test asserts that after InitBuildHandlers,
// the watcher's directory set includes the Go module root.
//
// The assertion reads the unexported `*devwatch.DevWatch.directories`
// field via reflection+unsafe — devwatch ships no public getter, and
// adding one is out of scope for this bug.
func TestWatcher_IncludesModuleRoot_NotJustStartDir(t *testing.T) {
	parent := t.TempDir()
	web := filepath.Join(parent, "web")
	sub := filepath.Join(parent, "sub")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "go.mod"),
		[]byte("module example.com/parent\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "client.go"),
		[]byte("//go:build wasm\n\npackage main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "ssr.go"),
		[]byte("//go:build !wasm\n\npackage sub\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := NewTestHandler(web)
	h.Tui = newUiMockTest()
	h.GitHandler = &MockGitClient{}
	h.Browser = &MockBrowser{}
	h.DB = &MockDB{data: make(map[string]string)}
	h.GoModHandler = &MockGoModHandler{}
	h.Logger = func(...any) {}

	h.InitBuildHandlers()

	dirs := watcherDirectories(t, h.Watcher)

	parentAbs, _ := filepath.Abs(parent)
	webAbs, _ := filepath.Abs(web)

	if !containsString(dirs, webAbs) {
		t.Errorf("watcher must include start dir %q; got %v", webAbs, dirs)
	}
	if !containsString(dirs, parentAbs) {
		t.Fatalf("watcher must include the Go module root %q so ssr.go in "+
			"sibling subpackages emits FS events; watched dirs: %v", parentAbs, dirs)
	}
}

// watcherDirectories reads the unexported `directories` slice from
// *devwatch.DevWatch via reflection+unsafe. Acceptable in test code:
// the bug is structural (which dirs are watched) and devwatch ships
// no public accessor.
func watcherDirectories(t *testing.T, w *devwatch.DevWatch) []string {
	t.Helper()
	v := reflect.ValueOf(w).Elem().FieldByName("directories")
	if !v.IsValid() {
		t.Fatalf("DevWatch has no `directories` field; devwatch internals changed")
	}
	v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
	out := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		out[i] = v.Index(i).String()
	}
	return out
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
