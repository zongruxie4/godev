package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/app"
)

func TestStart_SubdirectoryGuard(t *testing.T) {
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "project")
	subDir := filepath.Join(projectRoot, "sub")

	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test"), 0644)

	// Mock dependencies
	logger := app.NewLogger()
	ui := app.NewHeadlessTUI(logger.Logger)

	// Call Start from subdirectory
	result := app.Start(
		subDir,
		logger,
		ui,
		nil, // browser
		nil, // db
		make(chan bool), // exitChan
		nil, // serverFactory
		nil, // githubAuth
		nil, // gitHandler
		nil, // goModHandler
		true, // headless
		false, // clientMode
		nil, // onProjectReady
	)

	if result != false {
		t.Error("Start should return false when run from a subdirectory of a project")
	}
}
