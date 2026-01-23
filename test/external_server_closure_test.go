package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tinywasm/app"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/kvdb"
)

func TestExternalServerClosure(t *testing.T) {
	// Create a temp directory for the project
	tmpDir, err := os.MkdirTemp("", "tinywasm_regression_ext_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy go.mod
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module myapp\n\ngo 1.25.2\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create web/
	err = os.MkdirAll(filepath.Join(tmpDir, "web"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a dummy server.go in web/
	serverContent := `package main
import "net/http"
import "fmt"
func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello from external server")
	})
	http.ListenAndServe(":6061", nil)
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "web", "server.go"), []byte(serverContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Capture logs
	logger := func(messages ...any) {
		t.Log(messages...)
	}

	ExitChan := make(chan bool)

	finished := make(chan bool)
	go func() {
		mockBrowser := &MockBrowser{}
		mockDB, _ := kvdb.New(filepath.Join(tmpDir, ".env"), logger, app.NewMemoryStore())
		app.Start(tmpDir, logger, newUiMockTest(logger), mockBrowser, mockDB, ExitChan, devflow.NewMockGitHubAuth(), &MockGitClient{}) // Corrected app.Start call
		finished <- true
	}()

	// Wait for app.Handler
	h := app.WaitForActiveHandler(5 * time.Second)
	if h == nil {
		t.Fatal("Failed to get active app.Handler")
	}

	// Switch to External Mode
	t.Log("Switching to External Server Mode...")
	err = h.ServerHandler.SetExternalServerMode(true)
	if err != nil {
		t.Fatalf("Failed to switch to external mode: %v", err)
	}

	// Wait a bit to see if it finishes prematurely
	select {
	case <-finished:
		t.Fatal("App finished prematurely after switching to external mode!")
	case <-time.After(5 * time.Second):
		t.Log("App is still running in external mode")
		close(ExitChan)
		// Ensure it shuts down properly
		select {
		case <-finished:
			t.Log("App shut down cleanly")
		case <-time.After(2 * time.Second):
			t.Fatal("App took too long to shut down")
		}
	}
}
