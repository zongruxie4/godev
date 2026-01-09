package app

import (
	"os"
	"path/filepath"
)

// isInitializedProject checks if the current directory is an initialized Go project
// by verifying the existence of go.mod in the root directory.
func (h *handler) isInitializedProject() bool {
	goModPath := filepath.Join(h.config.RootDir(), "go.mod")
	_, err := os.Stat(goModPath)
	return err == nil
}
