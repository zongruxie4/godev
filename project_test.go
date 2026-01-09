package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsInitializedProject(t *testing.T) {
	tmp := t.TempDir()

	t.Run("ReturnsTrueIfGoModExists", func(t *testing.T) {
		h := &handler{
			config: NewConfig(tmp, nil),
		}

		// Create go.mod
		if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test"), 0644); err != nil {
			t.Fatalf("failed to create go.mod: %v", err)
		}

		if !h.isInitializedProject() {
			t.Error("expected isInitializedProject to return true when go.mod exists")
		}
	})

	t.Run("ReturnsFalseIfGoModDoesNotExist", func(t *testing.T) {
		tmp2 := t.TempDir()
		h := &handler{
			config: NewConfig(tmp2, nil),
		}

		if h.isInitializedProject() {
			t.Error("expected isInitializedProject to return false when go.mod does not exist")
		}
	})
}
