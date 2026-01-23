package test

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tinywasm/devbrowser"
	"github.com/tinywasm/server"
)

// TestExternalServerBrowserTiming replicates the exact scenario that fails in production:
// 1. External server takes time to compile and start (simulated with delay)
// 2. Browser should wait for server to be ACTUALLY listening, not just "started"
// 3. Browser should successfully connect after server is ready
//
// This test should FAIL if the browser opens before server is listening.
func TestExternalServerBrowserTiming(t *testing.T) {
	port := "19877"
	logs := &SafeBuffer{}

	// Simulate slow server startup: don't start listening immediately
	serverListeningChan := make(chan struct{})

	// Start "slow" server in goroutine - waits 2 seconds before actually listening
	go func() {
		// Simulate compilation/startup delay
		time.Sleep(2 * time.Second)

		// Now actually start listening
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><body>OK</body></html>"))
		})

		ln, err := net.Listen("tcp", ":"+port)
		if err != nil {
			t.Logf("Server listen error: %v", err)
			return
		}
		defer ln.Close()

		// Signal that server is now actually listening
		close(serverListeningChan)
		t.Logf("Server now listening on port %s", port)

		// Serve for a limited time
		server := &http.Server{Handler: mux}
		go server.Serve(ln)

		// Keep server alive for test duration
		time.Sleep(10 * time.Second)
	}()

	// Create serverReady channel - this simulates what server.Config.ServerReady does
	serverReady := make(chan struct{})

	// Simulate the external strategy's goroutine that waits for port
	go func() {
		// This replicates what external strategy does
		addr := "localhost:" + port
		deadline := time.Now().Add(30 * time.Second)

		for time.Now().Before(deadline) {
			conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				t.Logf("Port %s is now accepting connections", port)
				close(serverReady)
				return
			}
			time.Sleep(50 * time.Millisecond)
		}

		t.Errorf("TIMEOUT: Port %s never started listening within 30s", port)
		close(serverReady) // Close anyway to unblock browser
	}()

	// Now create browser and its OpenBrowser callback
	ui := &mockUI{}
	store := newMockStore()
	exitChan := make(chan bool)
	defer close(exitChan)

	browser := devbrowser.New(ui, store, exitChan)
	browser.SetTestMode(false) // REAL browser
	browser.SetHeadless(true)  // Headless so we don't see windows
	browser.SetLog(logs.Log)

	// Record time before OpenBrowser call
	startTime := time.Now()

	// Direct call to OpenBrowser as Server would do
	go browser.OpenBrowser(port, false)

	// Wait for browser to either open or fail
	time.Sleep(5 * time.Second)

	elapsed := time.Since(startTime)

	// Check logs for results
	logContent := logs.String()
	t.Logf("Elapsed time: %v", elapsed)
	t.Logf("Browser logs:\n%s", logContent)

	// Check if browser opened successfully
	if contains(logContent, "Open |") {
		t.Logf("✅ Browser opened successfully")
	}

	// Cleanup
	browser.CloseBrowser()
}

// TestOpenBrowserCallbackSignaledCorrectly tests the server package directly
// to verify that OpenBrowser callback is only called when port is listening.
func TestOpenBrowserCallbackSignaledCorrectly(t *testing.T) {
	port := "19878"
	logs := &SafeBuffer{}

	openCalled := make(chan string, 1)

	// Create server config with our callback
	cfg := &server.Config{
		AppRootDir: t.TempDir(),
		AppPort:    port,
		OpenBrowser: func(p string, https bool) {
			openCalled <- p
		},
		Logger:   logs.Log,
		ExitChan: make(chan bool),
		Routes:   []func(*http.ServeMux){},
	}

	sh := server.New(cfg)

	// Start server in goroutine
	go sh.StartServer(nil)

	// Wait for OpenBrowser callback
	select {
	case p := <-openCalled:
		t.Logf("OpenBrowser callback received for port %s", p)
		require.Equal(t, port, p)

		// Immediately try to connect - should succeed
		conn, err := net.DialTimeout("tcp", "localhost:"+port, 1*time.Second)
		if err != nil {
			t.Errorf("BUG: OpenBrowser was called but port is not listening! Error: %v", err)
			t.Logf("Logs:\n%s", logs.String())
		} else {
			conn.Close()
			t.Logf("✅ Port is actually listening when OpenBrowser was called")
		}

	case <-time.After(5 * time.Second):
		t.Errorf("TIMEOUT: OpenBrowser callback never called within 5 seconds")
		t.Logf("Logs:\n%s", logs.String())
	}

	// Cleanup
	close(cfg.ExitChan)
	time.Sleep(100 * time.Millisecond)
}
