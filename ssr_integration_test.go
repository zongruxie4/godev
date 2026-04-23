package app

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/mcp"
	"sync"
	"net/http"
)

func TestSSRLoadOnInit(t *testing.T) {
	root := t.TempDir()

	// Create a dummy module with ssr.go
	moduleDir := filepath.Join(root, "mymodule")
	os.MkdirAll(moduleDir, 0755)

	ssrContent := `
//go:build !wasm
package mymodule
func RenderCSS() string { return ".my-class { color: red; }" }
`
	os.WriteFile(filepath.Join(moduleDir, "ssr.go"), []byte(ssrContent), 0644)

	// Mock Handler
	h := &Handler{
		RootDir: root,
		Config:  NewConfig(root, nil),
		Tui:     &mockTui{},
		DB:      &mockDB{},
		serverFactory: func(exitChan chan bool, ui TuiInterface, browser BrowserInterface) ServerInterface {
			return &mockServer{}
		},
	}
	h.GoModHandler = &mockGoMod{
		replacePaths: []devflow.ReplaceEntry{{LocalPath: moduleDir}},
	}

	h.InitBuildHandlers()

	h.AssetsHandler.SetListModulesFn(func(rootDir string) ([]string, error) {
		return []string{moduleDir}, nil
	})

	// Manual call to ensure it's loaded for test
	err := h.AssetsHandler.LoadSSRModules()
	if err != nil {
		t.Fatalf("LoadSSRModules failed: %v", err)
	}

	if !h.AssetsHandler.ContainsCSS(".my-class") {
		t.Errorf("Expected CSS to contain '.my-class'")
	}
}

func TestSSRHotReload(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "mymodule")
	os.MkdirAll(moduleDir, 0755)

	os.WriteFile(filepath.Join(moduleDir, "ssr.go"), []byte(`
//go:build !wasm
package mymodule
func RenderCSS() string { return ".v1 { color: red; }" }
`), 0644)

	h := &Handler{
		RootDir: root,
		Config:  NewConfig(root, nil),
		Tui:     &mockTui{},
		DB:      &mockDB{},
		Browser: &mockBrowser{},
		serverFactory: func(exitChan chan bool, ui TuiInterface, browser BrowserInterface) ServerInterface {
			return &mockServer{}
		},
	}
	h.GoModHandler = &mockGoMod{
		replacePaths: []devflow.ReplaceEntry{{LocalPath: moduleDir}},
	}

	h.InitBuildHandlers()

	h.AssetsHandler.SetListModulesFn(func(rootDir string) ([]string, error) {
		return []string{moduleDir}, nil
	})

	// Initial load
	h.AssetsHandler.LoadSSRModules()

	if !h.AssetsHandler.ContainsCSS(".v1") {
		t.Errorf("Expected CSS to contain '.v1'")
	}

	// Update ssr.go
	os.WriteFile(filepath.Join(moduleDir, "ssr.go"), []byte(`
//go:build !wasm
package mymodule
func RenderCSS() string { return ".v2 { color: blue; }" }
`), 0644)

	// Trigger hot reload via the callback we wired
	h.GoModHandler.(*mockGoMod).onSSRFileChange(moduleDir)

	if !h.AssetsHandler.ContainsCSS(".v2") {
		t.Errorf("Expected CSS to contain '.v2' after hot reload")
	}
	if h.AssetsHandler.ContainsCSS(".v1") {
		t.Errorf("Expected CSS NOT to contain '.v1' after hot reload")
	}
}

// Mocks needed for the test
type mockTui struct{}
func (m *mockTui) NewTabSection(name, desc string) any { return nil }
func (m *mockTui) AddHandler(h any, color string, section any) {}
func (m *mockTui) Start(syncWaitGroup ...any) {}
func (m *mockTui) RefreshUI() {}
func (m *mockTui) ReturnFocus() error { return nil }
func (m *mockTui) SetActiveTab(section any) {}
func (m *mockTui) GetHandlerStates() []byte { return nil }
func (m *mockTui) DispatchAction(key, value string) bool { return false }

type mockDB struct{}
func (m *mockDB) Get(key string) (string, error) { return "", nil }
func (m *mockDB) Set(key, value string) error { return nil }
func (m *mockDB) Delete(key string) error { return nil }
func (m *mockDB) Close() error { return nil }
func (m *mockDB) Keys() ([]string, error) { return nil, nil }

type mockGoMod struct {
	replacePaths []devflow.ReplaceEntry
	onSSRFileChange func(string)
}
func (m *mockGoMod) GetReplacePaths() ([]devflow.ReplaceEntry, error) { return m.replacePaths, nil }
func (m *mockGoMod) SetLog(log func(...any)) {}
func (m *mockGoMod) SetFolderWatcher(w devflow.FolderWatcher) {}
func (m *mockGoMod) SetOnSSRFileChange(fn func(string)) { m.onSSRFileChange = fn }
func (m *mockGoMod) NewFileEvent(fileName, extension, filePath, event string) error { return nil }
func (m *mockGoMod) Name() string { return "GOMOD" }
func (m *mockGoMod) SupportedExtensions() []string { return nil }
func (m *mockGoMod) MainInputFileRelativePath() string { return "" }
func (m *mockGoMod) UnobservedFiles() []string { return nil }
func (m *mockGoMod) SetRootDir(path string) {}

type mockBrowser struct{}
func (m *mockBrowser) Reload() error { return nil }
func (m *mockBrowser) OpenBrowser(port string, https bool) {}
func (m *mockBrowser) SetLog(f func(message ...any)) {}
func (m *mockBrowser) GetLog() func(message ...any) { return nil }
func (m *mockBrowser) GetMCPTools() []mcp.Tool { return nil }

type mockServer struct{}
func (m *mockServer) StartServer(wg *sync.WaitGroup) {}
func (m *mockServer) StopServer() error { return nil }
func (m *mockServer) RestartServer() error { return nil }
func (m *mockServer) NewFileEvent(fileName, extension, filePath, event string) error { return nil }
func (m *mockServer) UnobservedFiles() []string { return nil }
func (m *mockServer) SupportedExtensions() []string { return nil }
func (m *mockServer) MainInputFileRelativePath() string { return "" }
func (m *mockServer) Name() string { return "SERVER" }
func (m *mockServer) Label() string { return "" }
func (m *mockServer) Value() string { return "" }
func (m *mockServer) Change(v string) {}
func (m *mockServer) RefreshUI() {}
func (m *mockServer) RegisterRoutes(fn func(*http.ServeMux)) {}
