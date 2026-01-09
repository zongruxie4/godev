package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/devflow"
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

func TestIsDirectoryEmpty(t *testing.T) {
	tmp := t.TempDir()
	h := &handler{
		config: NewConfig(tmp, nil),
	}

	t.Run("ReturnsTrueIfEmpty", func(t *testing.T) {
		if !h.isDirectoryEmpty() {
			t.Error("expected isDirectoryEmpty to return true for empty dir")
		}
	})

	t.Run("ReturnsFalseIfNotEmpty", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(tmp, "file.txt"), []byte("data"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if h.isDirectoryEmpty() {
			t.Error("expected isDirectoryEmpty to return false for non-empty dir")
		}
	})
}

func TestCanGenerateDefaultWasmClient(t *testing.T) {
	tmpParent := t.TempDir()
	tmpCurrent := filepath.Join(tmpParent, "current")
	os.Mkdir(tmpCurrent, 0755)

	git, _ := devflow.NewGit()
	gh, _ := devflow.NewGo(git)
	gh.SetRootDir(tmpCurrent)

	h := &handler{
		config:    NewConfig(tmpCurrent, nil),
		goHandler: gh,
	}

	t.Run("ReturnsFalseIfNotEmpty", func(t *testing.T) {
		os.WriteFile(filepath.Join(tmpCurrent, "file.txt"), []byte("data"), 0644)
		if h.canGenerateDefaultWasmClient() {
			t.Error("expected false because dir is not empty")
		}
		os.Remove(filepath.Join(tmpCurrent, "file.txt"))
	})

	t.Run("ReturnsFalseIfEmptyButNoGoMod", func(t *testing.T) {
		if h.canGenerateDefaultWasmClient() {
			t.Error("expected false because no go.mod in current or parent")
		}
	})

	t.Run("ReturnsFalseIfGoModInCurrent", func(t *testing.T) {
		path := filepath.Join(tmpCurrent, "go.mod")
		os.WriteFile(path, []byte("module test"), 0644)
		defer os.Remove(path)

		if h.canGenerateDefaultWasmClient() {
			t.Error("expected false because go.mod makes directory not empty")
		}
	})

	t.Run("ReturnsTrueIfEmptyAndGoModInParent", func(t *testing.T) {
		path := filepath.Join(tmpParent, "go.mod")
		os.WriteFile(path, []byte("module test-parent"), 0644)
		defer os.Remove(path)

		if !h.canGenerateDefaultWasmClient() {
			t.Error("expected true because empty and go.mod in parent")
		}
	})
}
