package godev

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerRecompileOnChange(t *testing.T) {
	// Do not run tests in parallel: they use a network port and Start().

	// Get the port early so we can check it
	port := freePort()
	t.Logf("Using port %s for test server", port)

	// Clean up any hanging processes from previous test runs
	checkRunningServerProcesses(t, "TEST START")
	checkPortUsage(t, port, "TEST START")
	killAllServerProcesses(t)
	checkRunningServerProcesses(t, "AFTER CLEANUP")
	checkPortUsage(t, port, "AFTER CLEANUP")

	tmp := t.TempDir()

	pwa := filepath.Join(tmp, "pwa")
	pwatheme := filepath.Join(pwa, "theme")
	require.NoError(t, os.MkdirAll(filepath.Join(pwa, "public"), 0755))
	require.NoError(t, os.MkdirAll(pwatheme, 0755))

	// export PORT so the compiled program uses our chosen port
	t.Setenv("PORT", port)

	// create minimal go.mod so `go build` inside the temp project works
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module temp\n\ngo 1.20\n"), 0644))

	// create theme files to replicate test/ structure
	require.NoError(t, os.WriteFile(filepath.Join(pwatheme, "index.html"), []byte("<h1>Test App</h1>"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(pwatheme, "main.js"), []byte("console.log('test app');"), 0644))

	// initial server source that returns v1 on /h
	initial := fmt.Sprintf(serverTemplate, port, "Server is running v1")
	updated := fmt.Sprintf(serverTemplate, port, "Server is running v2")

	mainPath := filepath.Join(pwa, "main.server.go")
	require.NoError(t, os.WriteFile(mainPath, []byte(initial), 0644))

	// write a small static file so file server has content
	require.NoError(t, os.WriteFile(filepath.Join(pwa, "public", "index.html"), []byte("ok"), 0644))

	// also place a go.mod inside pwa to make module resolution explicit
	require.NoError(t, os.WriteFile(filepath.Join(pwa, "go.mod"), []byte("module temp/pwa\n\ngo 1.20\n"), 0644))

	t.Logf("Created temp project structure in %s", tmp)
	t.Logf("main.server.go path: %s", mainPath)

	// Focused test flow:
	// - start godev Start() in goroutine
	// - wait ActiveHandler and watcher ready
	// - set SetWatcherBrowserReload to capture reload events
	// - write a broken main.server.go, then fix it
	// - assert that watcher triggered a restart (observed via logs or reload callback)

	var logMessages []string
	// collect logs but don't print them to test output unless we hit an error
	logger := func(v ...any) {
		msg := fmt.Sprint(v...)
		logMessages = append(logMessages, msg)
	}

	// Ensure port is free before starting
	if p, inUse := portListenerInfo(port); inUse {
		t.Logf("Port %s appears in use by: %s -- attempting to kill", port, p)
		// best-effort kill
		_ = killProcessUsingPort(port, t)
		time.Sleep(200 * time.Millisecond)
	}

	exitChan := make(chan bool, 1)
	startDone := make(chan bool, 1)

	go func() {
		defer func() { startDone <- true }()
		Start(tmp, logger, exitChan)
	}()

	// wait for ActiveHandler to be set
	waitUntil := time.Now().Add(8 * time.Second)
	for ActiveHandler == nil && time.Now().Before(waitUntil) {
		time.Sleep(50 * time.Millisecond)
	}
	if ActiveHandler == nil {
		t.Fatalf("ActiveHandler not set after Start()")
	}

	// install a reload callback to observe browser reloads
	reloadCount := 0
	SetWatcherBrowserReload(func() error {
		reloadCount++
		t.Logf("Browser reload called (%d)", reloadCount)
		return nil
	})

	// wait for watcher to be created
	waitWatcher := time.Now().Add(8 * time.Second)
	for ActiveHandler.watcher == nil && time.Now().Before(waitWatcher) {
		time.Sleep(50 * time.Millisecond)
	}
	if ActiveHandler.watcher == nil {
		t.Fatalf("watcher not created by AddSectionBUILD")
	}

	// allow system to stabilize
	time.Sleep(300 * time.Millisecond)

	// Ensure the initial server responds with v1 before we modify files below.
	// If this fails, show collected logs and fail immediately.
	if err := waitForServerContains(port, "v1", 10*time.Second); err != nil {
		t.Fatalf("initial server did not start with v1: %v; logs=%v", err, logMessages)
	}

	// write broken version
	broken := fmt.Sprintf(serverTemplate, port, "BROKEN")
	// inject a compile error
	broken = strings.Replace(broken, "fmt.Println", "fmt.rintf", 1)
	require.NoError(t, os.WriteFile(mainPath, []byte(broken), 0644))

	// give watcher time to process
	time.Sleep(800 * time.Millisecond)

	// fix file to updated content (v2)
	require.NoError(t, os.WriteFile(mainPath, []byte(updated), 0644))

	// wait for either logs or reload indicating restart
	deadline := time.Now().Add(12 * time.Second)
	sawRestart := false
	for time.Now().Before(deadline) {
		// check logs for restart indicators
		for _, m := range logMessages {
			if strings.Contains(m, "Go file modified") || strings.Contains(m, "restarting") || strings.Contains(m, "External server restarted") {
				sawRestart = true
				break
			}
		}
		if sawRestart || reloadCount > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if !sawRestart && reloadCount == 0 {
		// gather diagnostics
		ssOut, _ := runCmdOutput("ss", "-ltnp")
		psOut, _ := runCmdOutput("ps", "aux")
		files, _ := os.ReadDir(filepath.Join(tmp, "pwa"))
		var fileList []string
		for _, f := range files {
			fileList = append(fileList, f.Name())
		}
		t.Fatalf("did not observe server restart or browser reload after fixing code; logs=%v\nss:\n%s\nps:\n%s\nfiles:%v", logMessages, ssOut, psOut, fileList)
	}

	t.Logf("Observed restart: sawRestart=%v, reloadCount=%d", sawRestart, reloadCount)

	// stop the Start goroutines
	select {
	case exitChan <- true:
	default:
	}
	// wait for Start() to finish
	select {
	case <-startDone:
	case <-time.After(3 * time.Second):
		t.Log("Start() did not exit quickly after exit signal")
	}

	// Check if ActiveHandler was set and inspect its state
	if ActiveHandler != nil {
		t.Log("ActiveHandler was set")
		if ActiveHandler.config != nil {
			t.Logf("Config detected app types: %v", ActiveHandler.config.Types)
			t.Logf("Config web type: %v", ActiveHandler.config.WebType)
			t.Logf("Config web files folder: %s", ActiveHandler.config.GetWebFilesFolder())
			t.Logf("Config has web architecture: %v", ActiveHandler.config.HasWebArchitecture())
			t.Logf("Config root dir: %s", ActiveHandler.config.GetRootDir())
		} else {
			t.Log("ActiveHandler.config is nil")
		}
		if ActiveHandler.serverHandler != nil {
			t.Log("ServerHandler was created")
			// Check if the server can find the main.server.go file
			expectedPath := filepath.Join(ActiveHandler.config.GetRootDir(), ActiveHandler.config.GetWebFilesFolder(), "main.server.go")
			t.Logf("Expected server file path: %s", expectedPath)
			if _, err := os.Stat(expectedPath); err == nil {
				t.Log("✓ Server file exists at expected path")
			} else {
				t.Logf("✗ Server file not found at expected path: %v", err)
			}
		} else {
			t.Log("ServerHandler is nil")
		}
	} else {
		t.Log("ActiveHandler was not set")
	}

	// initial server verified above

	t.Log("server started successfully, now testing recompilation")
	// write updated source to trigger recompilation/restart
	require.NoError(t, os.WriteFile(mainPath, []byte(updated), 0644))

	t.Log("waiting for server restart (v2)")
	// wait for new version to appear
	require.NoError(t, waitForServerContains(port, "v2", 30*time.Second), "server did not restart with v2")

	t.Log("test completed successfully")
	// stop the Start goroutines
	select {
	case exitChan <- true:
		t.Log("Sent final exit signal")
	default:
		t.Log("Final exit channel blocked")
	}

	// Wait for graceful shutdown
	select {
	case <-startDone:
		t.Log("Start() completed gracefully")
	case <-time.After(5 * time.Second):
		t.Log("Start() did not complete within 5 seconds")
	}

	// Final cleanup - check for any remaining processes
	checkRunningServerProcesses(t, "TEST END")
	killAllServerProcesses(t)
}

// runCmdOutput executes a command and returns its combined output as string
func runCmdOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

// portListenerInfo tries to detect a process listening on the given port (tcp)
// returns the ss output line and true if found
func portListenerInfo(port string) (string, bool) {
	out, _ := runCmdOutput("ss", "-ltnp")
	if strings.Contains(out, ":"+port) {
		// return the matching line roughly
		lines := strings.Split(out, "\n")
		for _, l := range lines {
			if strings.Contains(l, ":"+port) {
				return l, true
			}
		}
	}
	return "", false
}

// killProcessUsingPort is a best-effort function to kill processes using port
func killProcessUsingPort(port string, t *testing.T) error {
	out, err := runCmdOutput("ss", "-ltnp")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, ":"+port) {
			// try to extract pid=
			idx := strings.Index(line, "pid=")
			if idx != -1 {
				rest := line[idx+4:]
				// pid may end with comma
				pid := rest
				for i, ch := range rest {
					if ch < '0' || ch > '9' {
						pid = rest[:i]
						break
					}
				}
				if pid != "" {
					_, _ = runCmdOutput("kill", "-9", pid)
					t.Logf("killed pid %s using port %s", pid, port)
				}
			}
		}
	}
	return nil
}
