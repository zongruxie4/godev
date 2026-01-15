package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsInitializedProject(t *testing.T) {
	t.Run("ReturnsTrueIfGoModExistsInCurrent", func(t *testing.T) {
		tmp := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test"), 0644); err != nil {
			t.Fatalf("failed to create go.mod: %v", err)
		}
		h := NewTestHandler(tmp)

		if !h.isInitializedProject() {
			t.Error("expected isInitializedProject to return true when go.mod exists in current")
		}
	})

	t.Run("ReturnsFalseIfGoModDoesNotExist", func(t *testing.T) {
		tmp := t.TempDir()
		h := NewTestHandler(tmp)

		if h.isInitializedProject() {
			t.Error("expected isInitializedProject to return false when go.mod does not exist")
		}
	})

	t.Run("ReturnsFalseIfGoModOnlyInParent", func(t *testing.T) {
		parent := t.TempDir()
		child := filepath.Join(parent, "subdir")
		os.Mkdir(child, 0755)
		os.WriteFile(filepath.Join(parent, "go.mod"), []byte("module test"), 0644)

		h := NewTestHandler(child)

		// isInitializedProject should return FALSE because go.mod is not in current dir
		if h.isInitializedProject() {
			t.Error("expected isInitializedProject to return false when go.mod only in parent")
		}
	})
}

func TestIsPartOfProject(t *testing.T) {
	t.Run("ReturnsTrueIfGoModInCurrent", func(t *testing.T) {
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test"), 0644)
		h := NewTestHandler(tmp)

		if !h.isPartOfProject() {
			t.Error("expected isPartOfProject to return true when go.mod in current")
		}
	})

	t.Run("ReturnsTrueIfGoModInParent", func(t *testing.T) {
		parent := t.TempDir()
		child := filepath.Join(parent, "subdir")
		os.Mkdir(child, 0755)
		os.WriteFile(filepath.Join(parent, "go.mod"), []byte("module test"), 0644)

		h := NewTestHandler(child)

		if !h.isPartOfProject() {
			t.Error("expected isPartOfProject to return true when go.mod in parent")
		}
	})

	t.Run("ReturnsFalseIfNoGoMod", func(t *testing.T) {
		tmp := t.TempDir()
		h := NewTestHandler(tmp)

		if h.isPartOfProject() {
			t.Error("expected isPartOfProject to return false when no go.mod")
		}
	})
}

func TestIsDirectoryEmpty(t *testing.T) {
	t.Run("ReturnsTrueIfEmpty", func(t *testing.T) {
		tmp := t.TempDir()
		h := NewTestHandler(tmp)

		if !h.isDirectoryEmpty() {
			t.Error("expected isDirectoryEmpty to return true for empty dir")
		}
	})

	t.Run("ReturnsFalseIfNotEmpty", func(t *testing.T) {
		tmp := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmp, "file.txt"), []byte("data"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		h := NewTestHandler(tmp)

		if h.isDirectoryEmpty() {
			t.Error("expected isDirectoryEmpty to return false for non-empty dir")
		}
	})
}

func TestCanGenerateDefaultWasmClient(t *testing.T) {
	t.Run("ReturnsFalseIfNotEmpty", func(t *testing.T) {
		parent := t.TempDir()
		os.WriteFile(filepath.Join(parent, "go.mod"), []byte("module test"), 0644)
		current := filepath.Join(parent, "current")
		os.Mkdir(current, 0755)
		os.WriteFile(filepath.Join(current, "file.txt"), []byte("data"), 0644)

		h := NewTestHandler(current)

		if h.canGenerateDefaultWasmClient() {
			t.Error("expected false because dir is not empty")
		}
	})

	t.Run("ReturnsFalseIfEmptyButNoGoMod", func(t *testing.T) {
		tmp := t.TempDir()
		h := NewTestHandler(tmp)

		if h.canGenerateDefaultWasmClient() {
			t.Error("expected false because no go.mod in current or parent")
		}
	})

	t.Run("ReturnsTrueIfGoModInCurrentMakesNotEmpty", func(t *testing.T) {
		// go.mod in current makes directory NOT empty, but allows generation if initialized
		tmp := t.TempDir()
		os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test"), 0644)
		h := NewTestHandler(tmp)

		if !h.canGenerateDefaultWasmClient() {
			t.Error("expected true because project is initialized")
		}
	})

	t.Run("ReturnsTrueIfEmptyAndGoModInParent", func(t *testing.T) {
		parent := t.TempDir()
		os.WriteFile(filepath.Join(parent, "go.mod"), []byte("module test-parent"), 0644)
		current := filepath.Join(parent, "current")
		os.Mkdir(current, 0755)

		h := NewTestHandler(current)

		if !h.canGenerateDefaultWasmClient() {
			t.Error("expected true because empty and go.mod in parent")
		}
	})
}
