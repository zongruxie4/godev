package app

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
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
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %q on %s (last status: %s)", substr, url, lastErr)
}

// startTestApp starts the app for testing and disables browser auto-start.
// Returns the handler and a cleanup function that should be deferred.
// Usage:
//
//	h, cleanup := startTestApp(t, tmpDir)
//	defer cleanup()
func startTestApp(t *testing.T, rootDir string) (*handler, func()) {
	exitChan := make(chan bool)

	logger := func(messages ...any) {
		var msg string
		for i, m := range messages {
			if i > 0 {
				msg += " "
			}
			msg += fmt.Sprint(m)
		}
		logIfVerbose(t, "LOG: %s", msg)
	}

	// Start tinywasm
	go Start(rootDir, logger, newUiMockTest(logger), exitChan)

	// Wait for handler initialization
	h := WaitForActiveHandler(5 * time.Second)
	if h == nil {
		t.Fatal("Failed to get active handler")
	}

	cleanup := func() {
		close(exitChan)
		SetActiveHandler(nil)
	}

	return h, cleanup
}
