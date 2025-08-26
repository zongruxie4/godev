package godev

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// checkRunningServerProcesses checks for any running main.server processes
func checkRunningServerProcesses(t *testing.T, label string) {
	t.Logf("🔍 %s: Checking for running main.server processes...", label)
	cmd := exec.Command("pgrep", "-f", "main.server")
	output, err := cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		t.Logf("🔍 %s: Found running main.server processes: %s", label, strings.TrimSpace(string(output)))

		// Get more details
		cmd2 := exec.Command("ps", "-aux")
		if output2, err2 := cmd2.CombinedOutput(); err2 == nil {
			lines := strings.Split(string(output2), "\n")
			for _, line := range lines {
				if strings.Contains(line, "main.server") {
					t.Logf("🔍 %s: Process details: %s", label, line)
				}
			}
		}
	} else {
		t.Logf("✅ %s: No main.server processes found (err: %v)", label, err)
	}
}

// checkPortUsage checks what process is using the specified port
func checkPortUsage(t *testing.T, port, label string) {
	t.Logf("🔍 %s: Checking port %s usage...", label, port)
	cmd := exec.Command("lsof", "-i", ":"+port)
	output, err := cmd.CombinedOutput()
	if err == nil && len(output) > 0 {
		t.Logf("🔍 %s: Port %s is in use:\n%s", label, port, string(output))
	} else {
		t.Logf("✅ %s: Port %s is free (err: %v)", label, port, err)
	}
}

// killAllServerProcesses kills any hanging main.server processes
func killAllServerProcesses(t *testing.T) {
	t.Log("🧹 Attempting to kill any hanging main.server processes...")
	cmd := exec.Command("pkill", "-f", "main.server")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Log("🧹 Killed any hanging main.server processes")
	} else {
		t.Logf("🧹 pkill result: %v, output: %s", err, string(output))
	}

	// Wait a bit for processes to die
	time.Sleep(500 * time.Millisecond)
}

func freePort() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "4430"
	}
	defer ln.Close()
	return fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
}
