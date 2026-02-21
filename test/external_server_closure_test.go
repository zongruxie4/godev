package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tinywasm/server"
)

func TestExternalServerClosure(t *testing.T) {
	// Create a temp directory for the project
	tmpDir := t.TempDir()

	// Create a dummy go.mod
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module myapp\n\ngo 1.25.2\n"), 0644)
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

	ctx := startTestApp(t, tmpDir)
	defer ctx.Cleanup()

	h := ctx.Handler

	// Switch to External Mode
	t.Log("Switching to External Server Mode...")
	err = h.Server.(*server.ServerHandler).SetExternalServerMode(true)
	if err != nil {
		t.Fatalf("Failed to switch to external mode: %v", err)
	}

	// To check for premature exit, we can monitor h.ExitChan
	select {
	case <-h.ExitChan:
		t.Fatal("App finished prematurely after switching to external mode!")
	case <-time.After(1 * time.Second):
		t.Log("App is still running in external mode")
	}
}
