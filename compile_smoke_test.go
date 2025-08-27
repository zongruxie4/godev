package godev

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCompileSmoke tries a simple go build to ensure server main compiles. Skipped in -short.
func TestCompileSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("smoke test - skipping in short mode")
	}

	// ensure 'go' exists
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go binary not available in PATH")
	}

	tmp := t.TempDir()
	pwa := filepath.Join(tmp, "pwa")
	if err := os.MkdirAll(pwa, 0755); err != nil {
		t.Fatalf("Failed to create pwa dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(pwa, "go.mod"), []byte("module temp/pwa\n\ngo 1.20\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	port := "8080"
	serverContent := fmt.Sprintf(serverTemplate, port, "Server is running v1")
	mainPath := filepath.Join(pwa, "main.server.go")
	if err := os.WriteFile(mainPath, []byte(serverContent), 0644); err != nil {
		t.Fatalf("Failed to create main.server.go: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", "main.server", "main.server.go")
	cmd.Dir = pwa
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Manual compilation failed: %v\nOutput: %s", err, string(output))
	}

	binPath := filepath.Join(pwa, "main.server")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("Binary not created: %v", err)
	}
}
