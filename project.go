package app

import (
	"os"
	"path/filepath"
)

// isInitializedProject checks if the current directory is an initialized Go project
// by verifying the existence of go.mod in the CURRENT directory only.
// Use this for features that should only work in the project root (.env, .gitignore, .vscode).
func (h *handler) isInitializedProject() bool {
	goModPath := filepath.Join(h.config.RootDir(), "go.mod")
	_, err := os.Stat(goModPath)
	return err == nil
}

// isPartOfProject checks if the current directory belongs to a Go project
// by verifying the existence of go.mod in the current directory OR its parent.
// Use this for features that should work in subdirectories (web/client.go, watcher).
func (h *handler) isPartOfProject() bool {
	return h.goHandler.ModExistsInCurrentOrParent()
}

// isDirectoryEmpty checks if the current directory has no files or subdirectories.
func (h *handler) isDirectoryEmpty() bool {
	entries, err := os.ReadDir(h.config.RootDir())
	if err != nil {
		return false
	}
	return len(entries) == 0
}

// canGenerateDefaultWasmClient returns true if:
// 1. Directory is completely empty
// 2. go.mod exists in current or parent directory (isPartOfProject)
func (h *handler) canGenerateDefaultWasmClient() bool {
	if !h.isDirectoryEmpty() {
		return false
	}
	return h.isPartOfProject()
}
