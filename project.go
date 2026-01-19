package app

import (
	"os"
	"path/filepath"
)

// IsInitializedProject checks if the current directory is an initialized Go project
// by verifying the existence of go.mod in the CURRENT directory only.
// Use this for features that should only work in the project root (.env, .gitignore, .vscode).
func (h *Handler) IsInitializedProject() bool {
	goModPath := filepath.Join(h.Config.RootDir, "go.mod")
	_, err := os.Stat(goModPath)
	return err == nil
}

// IsPartOfProject checks if the current directory belongs to a Go project
// by verifying the existence of go.mod in the current directory OR its parent.
// Use this for features that should work in subdirectories (web/client.go, Watcher).
func (h *Handler) IsPartOfProject() bool {
	return h.GoHandler.ModExistsInCurrentOrParent()
}

// IsDirectoryEmpty checks if the current directory has no files or subdirectories.
func (h *Handler) IsDirectoryEmpty() bool {
	entries, err := os.ReadDir(h.Config.RootDir)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

// CanGenerateDefaultWasmClient returns true if:
// 1. Project is initialized (go.mod exists in root) - allows adding client to new projects
// 2. OR Directory is empty AND parent has go.mod (submodule case)
func (h *Handler) CanGenerateDefaultWasmClient() bool {
	if h.IsInitializedProject() {
		return true
	}
	if !h.IsDirectoryEmpty() {
		return false
	}
	return h.IsPartOfProject()
}
