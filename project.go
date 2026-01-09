package app

import (
	"os"
)

// isInitializedProject checks if the current directory is an initialized Go project
// by verifying the existence of go.mod in the root directory or its parent.
func (h *handler) isInitializedProject() bool {
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
// 2. go.mod exists in current or parent directory
func (h *handler) canGenerateDefaultWasmClient() bool {
	if !h.isDirectoryEmpty() {
		return false
	}
	return h.goHandler.ModExistsInCurrentOrParent()
}
