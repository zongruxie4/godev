package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStart_SubdirectoryGuard(t *testing.T) {
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "project")
	subDir := filepath.Join(projectRoot, "sub")

	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644)

	// Direct subpackage (1 level under go.mod) must now be ALLOWED
	ctx := startTestApp(t, subDir)
	defer ctx.Cleanup()

	if strings.Contains(ctx.Logs.String(), "Directory Not Initialized") ||
		strings.Contains(ctx.Logs.String(), "Directorio No Inicializado") {
		t.Errorf("direct subpackage should be allowed, got: %s", ctx.Logs.String())
	}
}
