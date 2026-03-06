package app

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// isPortOpen checks if a TCP port is open on localhost
func isPortOpen(port string) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", port), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		conn.Close()
		return true
	}
	return false
}

// isDaemonVersionCurrent fetches /version from the running daemon and compares to ours.
// Returns true if versions match (daemon is current), false if stale or unreachable.
func isDaemonVersionCurrent(port, version string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:" + port + "/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var data struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false
	}
	return data.Version == version
}

// killDaemon sends SIGTERM to any running tinywasm daemon process.
func killDaemon() {
	exec.Command("pkill", "-f", "tinywasm -mcp").Run()
}

// waitForPortFree polls until the port is no longer listening (max 5s).
func waitForPortFree(port string) {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			return
		case <-ticker.C:
			if !isPortOpen(port) {
				return
			}
		}
	}
}

// waitForPortReady polls until the port is accepting connections (max 5s).
func waitForPortReady(port string) {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			fmt.Println("Timeout waiting for daemon to start")
			os.Exit(1)
		case <-ticker.C:
			if isPortOpen(port) {
				return
			}
		}
	}
}

// startDaemonProcess starts a detached daemon process
func startDaemonProcess(dir string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Start detached process
	cmd := exec.Command(exe, "-mcp")
	cmd.Dir = dir

	logPath := filepath.Join(dir, "tinywasm-daemon.log")
	logFile, err := os.Create(logPath)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	return cmd.Start()
}
