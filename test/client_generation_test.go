package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tinywasm/app"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/kvdb"
)

func TestClientGenerationInEmptyFolder(t *testing.T) {
	// Subdirectory execution is now rejected.
	// 1. Create a temporary directory structure:
	// /start (with go.mod) -> /start/subdir (empty, where we run app)
	tmpRoot := t.TempDir()

	// Create go.mod in root
	if err := os.WriteFile(filepath.Join(tmpRoot, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create and use subdir
	tmpDir := filepath.Join(tmpRoot, "subdir")
	if err := os.Mkdir(tmpDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 2. Start the app in this empty subdirectory - should return false/fail
	ExitChan := make(chan bool)
	logs := &SafeBuffer{}
	ui := newUiMockTest(logs.Log)
	db, _ := kvdb.New(filepath.Join(tmpDir, ".env"), logs.Log, app.NewMemoryStore())

	result := app.Start(tmpDir, logs.Log, ui, &MockBrowser{}, db, ExitChan, nil, nil, &MockGitClient{}, devflow.NewGoModHandler(), false, false, nil)

	if result != false {
		t.Errorf("Expected Start to return false for subdirectory execution, got true")
	}

	if !strings.Contains(logs.String(), "Directorio No Inicializado") && !strings.Contains(logs.String(), "Directory Not Initialized") {
		t.Errorf("Expected error message about uninitialized directory, got: %s", logs.String())
	}
}
