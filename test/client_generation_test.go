package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClientGenerationInEmptyFolder(t *testing.T) {
	// Direct subpackage execution is now ALLOWED.
	// Structure: /root (go.mod) -> /root/subdir (empty, where we run app)
	tmpRoot := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpRoot, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tmpDir := filepath.Join(tmpRoot, "subdir")
	if err := os.Mkdir(tmpDir, 0755); err != nil {
		t.Fatal(err)
	}

	ctx := startTestApp(t, tmpDir)
	defer ctx.Cleanup()

	if strings.Contains(ctx.Logs.String(), "Directory Not Initialized") ||
		strings.Contains(ctx.Logs.String(), "Directorio No Inicializado") {
		t.Errorf("direct subpackage should be allowed, got rejection: %s", ctx.Logs.String())
	}
}
