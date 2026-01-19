package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/devflow"
)

// TestInstallAllCommands installs all commands from cmd/ directory
// Run with: go test -run TestInstallAllCommands
func TestInstallAllCommands(t *testing.T) {
	// Get project root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Tests are now in app/test, so navigate to parent for cmd/
	appRoot := filepath.Dir(cwd)
	cmdDir := filepath.Join(appRoot, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		t.Skipf("Skipping: cmd/ directory not found at %s: %v", cmdDir, err)
	}

	// Collect all subdirectories (commands)
	var commands []string
	for _, entry := range entries {
		if entry.IsDir() {
			commands = append(commands, entry.Name())
		}
	}

	if len(commands) == 0 {
		t.Skip("No commands found in cmd/ directory")
	}

	// Change to app root to run go install
	originalDir, _ := os.Getwd()
	if err := os.Chdir(appRoot); err != nil {
		t.Fatalf("Failed to change to app root: %v", err)
	}
	defer os.Chdir(originalDir)

	for _, cmd := range commands {
		output, err := devflow.RunCommand("go", "install", "./cmd/"+cmd)
		if err != nil {
			t.Fatalf("Failed to install %s: %v\nOutput: %s", cmd, err, output)
		}
	}

	// t.Log("✅ All commands installed")
}
