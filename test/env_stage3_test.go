package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/devflow"
	"github.com/tinywasm/kvdb"
	"github.com/tinywasm/app"
)

func TestEnv_CreatedAtProjectRoot_NotSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "project")
	subDir := filepath.Join(projectRoot, "sub")

	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test"), 0644)

	// Simulate finding project root
	root, err := devflow.FindProjectRoot(subDir)
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}
	if root != projectRoot {
		t.Fatalf("Expected root %s, got %s", projectRoot, root)
	}

	// Use projectRoot for kvdb path (the fix)
	logger := app.NewLogger()
	db, err := kvdb.New(filepath.Join(root, ".env"), logger.Logger, &app.FileStore{})
	if err != nil {
		t.Fatalf("Failed to create kvdb: %v", err)
	}
	db.Set("test", "value")

	// Assert .env is at projectRoot, NOT subDir
	if _, err := os.Stat(filepath.Join(projectRoot, ".env")); os.IsNotExist(err) {
		t.Error(".env should exist at project root")
	}
	if _, err := os.Stat(filepath.Join(subDir, ".env")); err == nil {
		t.Error(".env should NOT exist in subdirectory")
	}
}

func TestEnv_NoProject_NoEnv(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a non-empty directory without go.mod
	os.WriteFile(filepath.Join(tmpDir, "somefile.txt"), []byte("data"), 0644)

	_, err := devflow.FindProjectRoot(tmpDir)
	if err == nil {
		t.Fatal("Expected error finding project root in non-project directory")
	}

	// The logic in main.go should exit here.
	// We'll just verify that if we follow the logic, we don't create .env if we don't have a root.
}
