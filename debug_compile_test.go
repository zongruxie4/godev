package godev

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDebugCompile(t *testing.T) {
	tmp := t.TempDir()

	// Create pwa directory structure
	pwa := filepath.Join(tmp, "pwa")
	if err := os.MkdirAll(pwa, 0755); err != nil {
		t.Fatalf("Failed to create pwa dir: %v", err)
	}

	// Create go.mod
	if err := os.WriteFile(filepath.Join(pwa, "go.mod"), []byte("module temp/pwa\n\ngo 1.20\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create main.server.go with the same content as the test
	port := "8080"
	serverContent := fmt.Sprintf(serverTemplate, port, "Server is running v1")
	mainPath := filepath.Join(pwa, "main.server.go")
	if err := os.WriteFile(mainPath, []byte(serverContent), 0644); err != nil {
		t.Fatalf("Failed to create main.server.go: %v", err)
	}

	t.Logf("Created files in: %s", pwa)
	t.Logf("main.server.go content preview: %s", serverContent[:100])

	// Try manual compilation
	cmd := exec.Command("go", "build", "-o", "main.server", "main.server.go")
	cmd.Dir = pwa
	output, err := cmd.CombinedOutput()

	t.Logf("Command: %s", cmd.String())
	t.Logf("Working dir: %s", cmd.Dir)
	t.Logf("Output: %s", string(output))
	t.Logf("Error: %v", err)

	if err != nil {
		t.Fatalf("Manual compilation failed: %v", err)
	}

	// Check if binary was created
	binPath := filepath.Join(pwa, "main.server")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("Binary not created: %v", err)
	}

	t.Log("Manual compilation successful!")
}
