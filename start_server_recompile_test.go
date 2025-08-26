package godev

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

	"github.com/stretchr/testify/require"
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

const serverTemplate = `package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "%s" // Template variable
	}

	publicDir := "public" // Template variable

	// Get current working directory for debugging
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %%v", err)
	} else {
		log.Printf("Current working directory: %%s", wd)
	}

	// Check if public directory exists
	absPublicDir, err := filepath.Abs(publicDir)
	if err != nil {
		log.Printf("Error getting absolute path for public dir: %%v", err)
	} else {
		log.Printf("Public directory absolute path: %%s", absPublicDir)
	}

	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		log.Printf("WARNING: Public directory '%%s' does not exist!", publicDir)
	} else {
		log.Printf("Public directory '%%s' exists", publicDir)
	}

	// Serve static files
	fs := http.FileServer(http.Dir(publicDir))

	// Use a dedicated ServeMux so we can pass it to an http.Server
	mux := http.NewServeMux()
	mux.Handle("/", fs)

	// Health check endpoint
	mux.HandleFunc("/h", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "%s")
	})

	// Create http.Server with Addr and Handler set
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	fmt.Printf("Server port %%s — Servin static files from: %%s\n", port, publicDir)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
`

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

	// Verify the file was created and check its content
	if content, err := os.ReadFile(mainPath); err != nil {
		t.Fatalf("Failed to read main.server.go: %v", err)
	} else {
		t.Logf("main.server.go content length: %d bytes", len(content))
		if strings.Contains(string(content), "Server is running v1") {
			t.Log("✓ main.server.go contains expected v1 response")
		} else {
			t.Log("✗ main.server.go does not contain v1 response")
		}
	}

	// Test manual compilation to verify our server code works
	t.Log("🔍 Testing manual compilation before starting godev...")
	cmd := exec.Command("go", "build", "-o", "main.server", "main.server.go")
	cmd.Dir = filepath.Join(tmp, "pwa")
	output, compileErr := cmd.CombinedOutput()
	if compileErr != nil {
		t.Logf("❌ Manual compilation failed: %v", compileErr)
		t.Logf("❌ Compilation output: %s", string(output))
	} else {
		t.Logf("✅ Manual compilation successful")

		// Check if binary was created
		binPath := filepath.Join(tmp, "pwa", "main.server")
		if _, err := os.Stat(binPath); err == nil {
			t.Logf("✅ Binary created at: %s", binPath)

			// Test run the binary briefly to verify it works
			runCmd := exec.Command("./main.server")
			runCmd.Dir = filepath.Join(tmp, "pwa")
			runCmd.Env = append(os.Environ(), "PORT="+port)

			// Start the process
			err := runCmd.Start()
			if err != nil {
				t.Logf("❌ Failed to start manual server: %v", err)
			} else {
				t.Logf("✅ Manual server started")

				// Give it a moment to start
				time.Sleep(1 * time.Second)

				// Test if it responds correctly
				testURL := "http://127.0.0.1:" + port + "/"
				if resp, err := http.Get(testURL); err == nil {
					defer resp.Body.Close()
					body, _ := io.ReadAll(resp.Body)
					t.Logf("✅ Manual server responds: %d, body: %q", resp.StatusCode, string(body))

					// Test the /h endpoint
					testURL2 := "http://127.0.0.1:" + port + "/h"
					if resp2, err2 := http.Get(testURL2); err2 == nil {
						defer resp2.Body.Close()
						body2, _ := io.ReadAll(resp2.Body)
						t.Logf("✅ Manual server /h responds: %d, body: %q", resp2.StatusCode, string(body2))
					} else {
						t.Logf("❌ Manual server /h not responding: %v", err2)
					}
				} else {
					t.Logf("❌ Manual server not responding: %v", err)
				}

				// Stop the manual process
				if runCmd.Process != nil {
					runCmd.Process.Kill()
					runCmd.Wait()
				}
			}
		} else {
			t.Logf("❌ Binary not found at: %s", binPath)
		}
	} // Check if the Start function is working at all
	var logMessages []string
	logger := func(v ...any) {
		msg := fmt.Sprint(v...)
		logMessages = append(logMessages, msg)
		t.Log("[GODEV]", msg)
	}

	exitChan := make(chan bool, 1)

	// Start in a separate goroutine and monitor it
	startDone := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Start() panicked: %v", r)
			}
			startDone <- true
		}()
		Start(tmp, logger, exitChan)
	}()

	// give more time for the system to initialize
	t.Log("Waiting for Start() to initialize...")
	time.Sleep(5 * time.Second)

	t.Logf("Collected %d log messages:", len(logMessages))
	for i, msg := range logMessages {
		t.Logf("  [%d] %s", i, msg)
	}

	// Test manual compilation to diagnose issues
	t.Log("🔍 Testing manual compilation...")

	compileCmd := exec.Command("go", "build", "-o", "main.server", "main.server.go")
	compileCmd.Dir = filepath.Join(tmp, "pwa")
	compileOutput, compileError := compileCmd.CombinedOutput()
	if compileError != nil {
		t.Logf("❌ Manual compilation failed: %v", compileError)
		t.Logf("❌ Compilation output: %s", string(compileOutput))
	} else {
		t.Logf("✅ Manual compilation successful")

		// Check if binary was created
		binPath := filepath.Join(tmp, "pwa", "main.server")
		if _, err := os.Stat(binPath); err == nil {
			t.Logf("✅ Binary created at: %s", binPath)

			// Try to run it manually
			runCmd := exec.Command("./main.server")
			runCmd.Dir = filepath.Join(tmp, "pwa")
			runCmd.Env = append(os.Environ(), "PORT="+port)

			go func() {
				runOutput, runErr := runCmd.CombinedOutput()
				if runErr != nil {
					t.Logf("❌ Manual run failed: %v", runErr)
					t.Logf("❌ Run output: %s", string(runOutput))
				} else {
					t.Logf("✅ Manual run output: %s", string(runOutput))
				}
			}()

			// Give the manual server time to start
			time.Sleep(2 * time.Second)

			// Test if manual server responds
			testURL := "http://127.0.0.1:" + port + "/"
			if resp, err := http.Get(testURL); err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				t.Logf("✅ Manual server responds: %d, body: %q", resp.StatusCode, string(body))
			} else {
				t.Logf("❌ Manual server not responding: %v", err)
			}

			// Stop the manual process
			if runCmd.Process != nil {
				runCmd.Process.Kill()
			}
		} else {
			t.Logf("❌ Binary not found at: %s", binPath)
		}
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

	t.Log("waiting for initial server (v1)")
	// Test manual compilation to diagnose issues BEFORE waiting for the godev server
	t.Log("🔍 Testing manual compilation...")
	cmd2 := exec.Command("go", "build", "-o", "main.server", "main.server.go")
	cmd2.Dir = filepath.Join(tmp, "pwa")
	output2, compileErr2 := cmd2.CombinedOutput()
	if compileErr2 != nil {
		t.Logf("❌ Manual compilation failed: %v", compileErr2)
		t.Logf("❌ Compilation output: %s", string(output2))
	} else {
		t.Logf("✅ Manual compilation successful")

		// Check if binary was created
		binPath := filepath.Join(tmp, "pwa", "main.server")
		if _, err := os.Stat(binPath); err == nil {
			t.Logf("✅ Binary created at: %s", binPath)

			// Try to run it manually for a brief test
			runCmd := exec.Command("./main.server")
			runCmd.Dir = filepath.Join(tmp, "pwa")
			runCmd.Env = append(os.Environ(), "PORT="+port)

			// Create a timeout context for manual test
			go func() {
				time.Sleep(3 * time.Second)
				if runCmd.Process != nil {
					runCmd.Process.Kill()
				}
			}()

			runOutput, runErr := runCmd.CombinedOutput()
			if runErr != nil {
				t.Logf("❌ Manual run failed: %v", runErr)
				t.Logf("❌ Run output: %s", string(runOutput))
			} else {
				t.Logf("✅ Manual run output: %s", string(runOutput))
			}
		} else {
			t.Logf("❌ Binary not found at: %s", binPath)
		}
	}

	// wait for first version to be available with shorter timeout first
	err := waitForServerContains(port, "v1", 10*time.Second)
	if err != nil {
		t.Logf("Server startup failed: %v", err)
		t.Logf("Final log messages count: %d", len(logMessages))

		// Try to manually test different endpoints to understand what's running
		endpoints := []string{"/", "/h", "/health"}
		for _, endpoint := range endpoints {
			url := "http://127.0.0.1:" + port + endpoint
			if resp, httpErr := http.Get(url); httpErr == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				t.Logf("GET %s returned status %d, body: %q", endpoint, resp.StatusCode, string(body))
			} else {
				t.Logf("GET %s failed: %v", endpoint, httpErr)
			}
		}

		// Try to gracefully shutdown and see what happens
		select {
		case exitChan <- true:
			t.Log("Sent exit signal")
		default:
			t.Log("Exit channel blocked")
		}

		// Wait a bit for shutdown
		select {
		case <-startDone:
			t.Log("Start() completed")
		case <-time.After(2 * time.Second):
			t.Log("Start() still running after exit signal")
		}

		require.NoError(t, err, "initial server did not start with v1")
	}

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
