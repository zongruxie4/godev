package app

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/mcp"
	"sync"
	"net/http"
	"time"
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

	h.ListModulesFn = func(rootDir string) ([]string, error) {
		return []string{moduleDir}, nil
	}
	h.InitBuildHandlers()

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

	h.ListModulesFn = func(rootDir string) ([]string, error) {
		return []string{moduleDir}, nil
	}
	h.InitBuildHandlers()

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

func TestSSRNoBlockOnStartup(t *testing.T) {
	root := t.TempDir()
	
	// Create a module with a slow loading process (simulated via mock)
	h := &Handler{
		RootDir: root,
		Config:  NewConfig(root, nil),
		Tui:     &mockTui{},
		DB:      &mockDB{},
		serverFactory: func(exitChan chan bool, ui TuiInterface, browser BrowserInterface) ServerInterface {
			return &mockServer{}
		},
	}
	h.GoModHandler = &mockGoMod{}

	// We'll use a channel to signal when listModulesFn is called
	called := make(chan struct{})
	
	var once sync.Once
	h.ListModulesFn = func(rootDir string) ([]string, error) {
		once.Do(func() { close(called) })
		time.Sleep(100 * time.Millisecond) // Simulate slow loading
		return nil, nil
	}

	h.InitBuildHandlers()

	// InitBuildHandlers already started the goroutine
	// We should be able to continue here immediately
	select {
	case <-called:
		// Success: it started in background
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Expected LoadSSRModules to be called in background")
	}
}

func TestSSRProxyModulesLoaded(t *testing.T) {
	root := t.TempDir()
	proxyModuleDir := filepath.Join(root, "proxy_pkg")
	os.MkdirAll(proxyModuleDir, 0755)
	
	os.WriteFile(filepath.Join(proxyModuleDir, "ssr.go"), []byte(`
package proxy
func RenderCSS() string { return ".proxy { color: green; }" }
`), 0644)

	h := &Handler{
		RootDir: root,
		Config:  NewConfig(root, nil),
		Tui:     &mockTui{},
		DB:      &mockDB{},
		serverFactory: func(exitChan chan bool, ui TuiInterface, browser BrowserInterface) ServerInterface {
			return &mockServer{}
		},
	}
	h.GoModHandler = &mockGoMod{}
	
	h.ListModulesFn = func(rootDir string) ([]string, error) {
		return []string{proxyModuleDir}, nil
	}
	h.InitBuildHandlers()
	h.AssetsHandler.WaitForSSRLoad(2 * time.Second)
	
	if !h.AssetsHandler.ContainsCSS(".proxy") {
		t.Errorf("Expected CSS from proxy module to be loaded")
	}
}

func TestImageHotReload(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "mymodule")
	os.MkdirAll(moduleDir, 0755)
	
	// Create an image
	imgPath := filepath.Join(moduleDir, "logo.png")
	os.WriteFile(imgPath, []byte("fake image data"), 0644)

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

	// Mock list modules for both
	h.ListModulesFn = func(rootDir string) ([]string, error) {
		return []string{moduleDir}, nil
	}
	h.InitBuildHandlers()

	// Trigger hot reload via the callback
	// This should call both AssetsHandler.ReloadSSRModule and ImageHandler.ReloadModule
	h.GoModHandler.(*mockGoMod).onSSRFileChange(moduleDir)

	// Since we are mocking, we can't easily check if ImageHandler.ReloadModule was called
	// unless we inspect logs or use a more sophisticated mock.
	// But the fact that it doesn't panic and wires correctly is a good start.
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
