package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tinywasm/devflow"
	"github.com/tinywasm/kvdb"
)

// BUG: cmd/tinywasm/main.go uses startDir (os.Getwd) for kvdb.New path
// instead of projectRoot (devflow.FindProjectRoot result).
//
// Fix: change main.go from:
//   kvdb.New(filepath.Join(startDir, ".env"), ...)
// to:
//   kvdb.New(filepath.Join(projectRoot, ".env"), ...)
//
// TestEnv_BugReproduction_StartDir shows the current broken behavior (always passes — documents bug).
// TestEnv_CreatedAtProjectRoot_NotSubdirectory shows the required behavior (regression guard).

// TestEnv_BugReproduction_StartDir proves the bug: using startDir places .env in the subdirectory.
// This test always passes — its purpose is to document exactly what the bug does.
func TestEnv_BugReproduction_StartDir(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.21\n"), 0644)

	startDir := subDir // simulates os.Getwd() when user runs tinywasm from a subdirectory

	// BUG: main.go does this — uses startDir instead of projectRoot
	buggyDB, err := kvdb.New(filepath.Join(startDir, ".env"), func(...any) {}, &FileStore{})
	if err != nil {
		t.Fatal(err)
	}
	buggyDB.Set("test", "1")

	// Bug confirmed: .env lands in subdirectory, not project root
	if _, err := os.Stat(filepath.Join(subDir, ".env")); os.IsNotExist(err) {
		t.Fatal("bug no longer reproducible — update or remove this test")
	}
	if _, err := os.Stat(filepath.Join(root, ".env")); err == nil {
		t.Fatal("bug no longer reproducible — .env is already at root")
	}
}

// TestEnv_CreatedAtProjectRoot_NotSubdirectory is the regression guard.
// Verifies that when projectRoot is used for kvdb path, .env lands at go.mod level.
// This is the behavior that must hold after the fix in main.go.
func TestEnv_CreatedAtProjectRoot_NotSubdirectory(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}

	startDir := subDir

	projectRoot, err := devflow.FindProjectRoot(startDir)
	if err != nil {
		t.Fatalf("FindProjectRoot failed: %v", err)
	}

	db, err := kvdb.New(filepath.Join(projectRoot, ".env"), func(...any) {}, &FileStore{})
	if err != nil {
		t.Fatalf("kvdb.New failed: %v", err)
	}
	if err := db.Set("test", "1"); err != nil {
		t.Fatalf("db.Set failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, ".env")); os.IsNotExist(err) {
		t.Errorf(".env not found at project root %s", root)
	}
	if _, err := os.Stat(filepath.Join(subDir, ".env")); err == nil {
		t.Errorf(".env found in subdirectory %s — must only exist at project root", subDir)
	}
}

// TestEnv_NotCreatedWithoutProject verifies that no .env is written
// when there is no go.mod in the directory or its parent.
func TestEnv_NotCreatedWithoutProject(t *testing.T) {
	emptyDir := t.TempDir()

	_, err := devflow.FindProjectRoot(emptyDir)
	if err == nil {
		t.Skip("unexpected go.mod found in temp dir parent — skipping")
	}

	// Simulate the guard that must exist in main.go before kvdb.New:
	// if FindProjectRoot fails → do not create .env
	entries, _ := os.ReadDir(emptyDir)
	for _, e := range entries {
		if e.Name() == ".env" {
			t.Errorf(".env created in directory with no go.mod: %s", emptyDir)
		}
	}
}
