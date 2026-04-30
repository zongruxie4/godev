package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitBuildHandlers_SSRMode_InMemory(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "mymodule")
	os.MkdirAll(moduleDir, 0755)

	// Create a valid project structure
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module testapp\ngo 1.25\n"), 0644)
	os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nimport _ \"testapp/mymodule\"\nfunc main() {}"), 0644)

	ssrGoPath := filepath.Join(moduleDir, "ssr.go")
	os.WriteFile(ssrGoPath, []byte(`
package mymodule
func RenderCSS() string { return ".v1 { color: red; }" }
`), 0644)

	TestMode = true
	h := &Handler{
		RootDir: root,
		Config:  NewConfig(root, nil),
		Tui:     &mockTui{},
		DB:      &mockDB{},
		Browser: &mockBrowser{},
	}
	h.SetServerFactory(func(exitChan chan bool, ui TuiInterface, browser BrowserInterface) ServerInterface {
		return &mockServer{}
	})
	h.GoModHandler = &mockGoMod{}
	h.ListModulesFn = func(rootDir string) ([]string, error) {
		return []string{moduleDir}, nil
	}

	h.InitBuildHandlers()

	// Initial load
	h.AssetsHandler.UpdateSSRModule("testapp/mymodule", ".v1 { color: red; }", "", "", nil)

	if !h.AssetsHandler.ContainsCSS(".v1") {
		t.Errorf("Expected CSS to contain '.v1'")
	}

	// Update module - Simulate hot reload
	h.AssetsHandler.UpdateSSRModule("testapp/mymodule", ".v2 { color: blue; }", "", "", nil)

	// Verify update
	if !h.AssetsHandler.ContainsCSS(".v2") {
		t.Errorf("Expected CSS to contain '.v2' after update")
	}

	// Verify NO DUPLICATES/STALE CSS
	// If it was duplicating, it would still contain .v1
	if h.AssetsHandler.ContainsCSS(".v1") {
		t.Errorf("Expected CSS NOT to contain '.v1' after update (should have been replaced)")
	}

	// Verify no files written to disk (in-memory mode)
	publicCSS := filepath.Join(root, "web/public/main.css")
	if _, err := os.Stat(publicCSS); err == nil {
		t.Errorf("Expected NO CSS file on disk in in-memory mode, but found %s", publicCSS)
	}
}
