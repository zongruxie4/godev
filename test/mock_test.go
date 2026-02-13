package test

import (
	"net/http"
	"sync"

	"github.com/tinywasm/devflow"
	"github.com/tinywasm/mcpserve"
)

type mockTUI struct {
	logger func(...any)
}

func newUiMockTest(opt ...any) *mockTUI {
	m := &mockTUI{}
	if len(opt) > 0 {
		if l, ok := opt[0].(func(...any)); ok {
			m.logger = l
		}
	}
	return m
}

func (m *mockTUI) NewTabSection(title, description string) any {
	return nil
}

func (m *mockTUI) AddHandler(handler any, color string, tabSection any) {
	// Mimic DevTUI's behavior: call SetLog if app.Handler implements Loggable
	type Loggable interface {
		Name() string
		SetLog(logger func(message ...any))
	}

	if loggable, ok := handler.(Loggable); ok {
		logFunc := func(message ...any) {
			if m.logger != nil {
				m.logger(message...)
			}
		}
		loggable.SetLog(logFunc)
	}
}

func (m *mockTUI) AddLogger(name string, enableTracking bool, color string, tabSection any) func(message ...any) {
	return func(messages ...any) {
		if m.logger != nil {
			m.logger(messages...)
		}
	}
}

func (m *mockTUI) Start(syncWaitGroup ...any) {
	if len(syncWaitGroup) > 0 {
		if wg, ok := syncWaitGroup[0].(*sync.WaitGroup); ok {
			defer wg.Done()
		}
	}
	// Mimic blocking behavior if needed, or just return.
	// Real TUI blocks?
	// If real TUI blocks, we should probably block until exit?
	// But mockTUI is simple.
	// If we just Done() and return, wg decrements.
	// app.Start continues waiting for others.
}

func (m *mockTUI) RefreshUI() {
	// no-op
}

func (m *mockTUI) ReturnFocus() error {
	return nil
}

func (m *mockTUI) SetActiveTab(section any) {
	// no-op
}

type MockBrowser struct {
	reloadCalls   int
	openCalls     int    // Track actual browser open attempts
	lastOpenPort  string // Track last port used for open
	lastOpenHttps bool   // Track last https flag used for open
	ReloadErr     error
	mu            sync.Mutex
	logFunc       func(message ...any)
}

func (m *MockBrowser) GetReloadCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reloadCalls
}

func (m *MockBrowser) GetOpenCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.openCalls
}

func (m *MockBrowser) GetLastOpenPort() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastOpenPort
}

func (m *MockBrowser) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadCalls++
	return m.ReloadErr
}

func (m *MockBrowser) OpenBrowser(port string, https bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.openCalls++
	m.lastOpenPort = port
	m.lastOpenHttps = https
	if m.logFunc != nil {
		m.logFunc("MockBrowser: OpenBrowser called with port", port, "https", https)
	}
}

func (m *MockBrowser) SetLog(f func(message ...any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logFunc = f
}

func (m *MockBrowser) GetMCPToolsMetadata() []mcpserve.ToolMetadata {
	return []mcpserve.ToolMetadata{}
}

type MockGitClient struct {
	SetRootDirCalls int
}

func (m *MockGitClient) SetRootDir(path string) {
	m.SetRootDirCalls++
}

func (m *MockGitClient) CheckRemoteAccess() error {
	return nil
}

func (m *MockGitClient) Push(message, tag string) (devflow.PushResult, error) {
	return devflow.PushResult{}, nil
}

func (m *MockGitClient) GetLatestTag() (string, error) {
	return "v0.0.0", nil
}

func (m *MockGitClient) SetLog(fn func(...any)) {
}

func (m *MockGitClient) SetShouldWrite(fn func() bool) {
}

func (m *MockGitClient) GitIgnoreAdd(entry string) error {
	return nil
}

func (m *MockGitClient) GetConfigUserName() (string, error) {
	return "Mock User", nil
}

func (m *MockGitClient) GetConfigUserEmail() (string, error) {
	return "mock@example.com", nil
}

func (m *MockGitClient) InitRepo(dir string) error {
	return nil
}

func (m *MockGitClient) Add() error {
	return nil
}

func (m *MockGitClient) Commit(message string) (bool, error) {
	return true, nil
}

func (m *MockGitClient) CreateTag(tag string) (bool, error) {
	return true, nil
}

func (m *MockGitClient) PushWithTags(tag string) (bool, error) {
	return true, nil
}

type MockGitHubClient struct {
	log func(...any)
}

func (m *MockGitHubClient) SetLog(fn func(...any)) {
	if fn != nil {
		m.log = fn
	}
}

func (m *MockGitHubClient) GetCurrentUser() (string, error) {
	return "mockuser", nil
}

func (m *MockGitHubClient) RepoExists(owner, name string) (bool, error) {
	return false, nil
}

func (m *MockGitHubClient) CreateRepo(owner, name, description, visibility string) error {
	if m.log != nil {
		m.log("MockGitHub: Created repo", owner+"/"+name)
	}
	return nil
}

func (m *MockGitHubClient) DeleteRepo(owner, name string) error {
	if m.log != nil {
		m.log("MockGitHub: Deleted repo", owner+"/"+name)
	}
	return nil
}

func (m *MockGitHubClient) IsNetworkError(err error) bool {
	return false
}

func (m *MockGitHubClient) GetHelpfulErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type MockServer struct {
	StartServerCalls   int
	StopServerCalls    int
	RestartServerCalls int
	RegisteredRoutes   int
}

func (m *MockServer) StartServer(wg *sync.WaitGroup) {
	m.StartServerCalls++
	if wg != nil {
		defer wg.Done()
	}
}

func (m *MockServer) StopServer() error {
	m.StopServerCalls++
	return nil
}

func (m *MockServer) RestartServer() error {
	m.RestartServerCalls++
	return nil
}

func (m *MockServer) NewFileEvent(fileName, extension, filePath, event string) error {
	return nil
}

func (m *MockServer) UnobservedFiles() []string {
	return []string{}
}

func (m *MockServer) SupportedExtensions() []string {
	return []string{}
}

func (m *MockServer) Name() string {
	return "mock-server"
}

func (m *MockServer) Label() string {
	return "Mock Server"
}

func (m *MockServer) Value() string {
	return ""
}

func (m *MockServer) Change(v string) error {
	return nil
}

func (m *MockServer) RefreshUI() {
}

func (m *MockServer) RegisterRoutes(fn func(*http.ServeMux)) {
	m.RegisteredRoutes++
}
