package test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinywasm/app"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/kvdb"
	"github.com/tinywasm/server"
)

// logIfVerbose prints test logs only when GOLITE_TEST_VERBOSE is set.
func logIfVerbose(t *testing.T, format string, args ...any) {
	if os.Getenv("GOLITE_TEST_VERBOSE") != "" {
		t.Logf(format, args...)
	}
}

// checkRunningServerProcesses checks for any running main.server processes
func checkRunningServerProcesses(t *testing.T, label string) {
	logIfVerbose(t, "🔍 %s: Checking for running main.server processes...", label)
	cmd := exec.Command("pgrep", "-f", "main.server")
	output, err := cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		logIfVerbose(t, "🔍 %s: Found running main.server processes: %s", label, strings.TrimSpace(string(output)))

		// Get more details
		cmd2 := exec.Command("ps", "-aux")
		if output2, err2 := cmd2.CombinedOutput(); err2 == nil {
			lines := strings.Split(string(output2), "\n")
			for _, line := range lines {
				if strings.Contains(line, "main.server") {
					logIfVerbose(t, "🔍 %s: Process details: %s", label, line)
				}
			}
		}
	} else {
		logIfVerbose(t, "✅ %s: No main.server processes found (err: %v)", label, err)
	}
}

// checkPortUsage checks what process is using the specified port
func checkPortUsage(t *testing.T, port, label string) {
	logIfVerbose(t, "🔍 %s: Checking port %s usage...", label, port)
	cmd := exec.Command("lsof", "-i", ":"+port)
	output, err := cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		logIfVerbose(t, "🔍 %s: Port %s is in use:\n%s", label, port, string(output))
	} else {
		logIfVerbose(t, "✅ %s: Port %s is free (err: %v)", label, port, err)
	}
}

// killAllServerProcesses kills any hanging main.server processes
func killAllServerProcesses(t *testing.T) {
	logIfVerbose(t, "🧹 Attempting to kill any hanging main.server processes...")
	cmd := exec.Command("pkill", "-f", "main.server")
	output, err := cmd.CombinedOutput()
	if err == nil {
		logIfVerbose(t, "🧹 Killed any hanging main.server processes")
	} else {
		logIfVerbose(t, "🧹 pkill result: %v, output: %s", err, string(output))
	}

	// Wait a bit for processes to die
	time.Sleep(500 * time.Millisecond)
}

func freePort() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "6060"
	}
	defer ln.Close()
	return fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
}

func waitForServerContains(port, substr string, timeout time.Duration) error {
	url := "http://127.0.0.1:" + port + "/h"
	deadline := time.Now().Add(timeout)
	lastErr := ""
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			content := string(b)
			if strings.Contains(content, substr) {
				return nil
			}
			lastErr = fmt.Sprintf("got response: %q, waiting for %q", content, substr)
		} else {
			lastErr = fmt.Sprintf("connection error: %v", err)
		}
		// Try to connect to verify server is actually running
		// This part of the original instruction was syntactically incorrect and
		// would break the function. It's omitted to maintain a working test helper.
		// The original logic for checking /h is retained.
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %q on %s (last status: %s)", substr, url, lastErr)
}

type TestContext struct {
	Handler    *app.Handler
	Browser    *MockBrowser
	GitHandler *MockGitClient
	Logs       *SafeBuffer
	DB         app.DB
	UI         app.TuiInterface
	Cleanup    func()
}


// startTestApp starts the app for testing and disables Browser auto-start.
// Returns a TestContext containing the handler, mocks, and a cleanup function.
// Accepts optional overrides for Browser, GitClient, GoModHandler, DB, or TuiInterface.
func startTestApp(t *testing.T, RootDir string, opts ...any) *TestContext {
	ExitChan := make(chan bool)
	logs := &SafeBuffer{}

	// 1. Establish the logger (merging logs.Log with any custom logger passed)
	var customLogger func(messages ...any)
	for _, o := range opts {
		if l, ok := o.(func(messages ...any)); ok {
			customLogger = l
			break
		}
	}
	logger := logs.Log
	if customLogger != nil {
		logger = func(messages ...any) {
			logs.Log(messages...)
			customLogger(messages...)
		}
	}

	// 2. Initialize ctx with ALL defaults using the logger
	ctx := &TestContext{
		Logs:       logs,
		Browser:    &MockBrowser{},
		GitHandler: &MockGitClient{},
		UI:         newUiMockTest(logger),
	}
	ctx.DB, _ = kvdb.New(filepath.Join(RootDir, ".env"), logger, app.NewMemoryStore())

	// 3. Apply overrides from variadic arguments
	var goModH devflow.GoModInterface
	for _, o := range opts {
		switch v := o.(type) {
		case *MockBrowser:
			ctx.Browser = v
		case *MockGitClient:
			ctx.GitHandler = v
		case app.DB:
			ctx.DB = v
		case app.TuiInterface:
			ctx.UI = v
		case devflow.GoModInterface:
			goModH = v
		case app.BrowserInterface:
			if mb, ok := v.(*MockBrowser); ok {
				ctx.Browser = mb
			}
		case devflow.GitClient:
			if mg, ok := v.(*MockGitClient); ok {
				ctx.GitHandler = mg
			}
		}
	}

	// Default GoMod if not provided
	if goModH == nil {
		goModH = devflow.NewGoModHandler()
	}

	// 4. Set environment variables for dynamic ports
	serverPort := freePort()
	mcpPort := freePort()

	// Ensure we don't collide with default ports if they are busy
	os.Setenv("PORT", serverPort)
	os.Setenv("TINYWASM_MCP_PORT", mcpPort)

	// Wait a bit for env to propagate if needed (usually instant)

	// Start the application
	factory := func() app.ServerInterface {
		return server.New().
			SetLogger(logger).
			SetExitChan(ExitChan).
			SetStore(ctx.DB).
			SetUI(ctx.UI).
			SetOpenBrowser(ctx.Browser.OpenBrowser).
			SetGitIgnoreAdd(ctx.GitHandler.GitIgnoreAdd)
	}

	appDone := make(chan struct{})
	go func() {
		app.Start(RootDir, logger, ctx.UI, ctx.Browser, ctx.DB, ExitChan, factory, devflow.NewMockGitHubAuth(), ctx.GitHandler, goModH)
		close(appDone)
	}()
	// Wait for handler registration
	h := app.WaitForActiveHandler(8 * time.Second)
	if h == nil {
		t.Fatal("Failed to get active app.Handler")
	}
	ctx.Handler = h

	ctx.Cleanup = func() {
		close(ExitChan)
		app.SetActiveHandler(nil)
		os.Unsetenv("PORT")
		os.Unsetenv("TINYWASM_MCP_PORT")
		select {
		case <-appDone:
		case <-time.After(5 * time.Second):
			// force-proceed after 5s even if goroutines are slow
		}
	}

	return ctx
}
