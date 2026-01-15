package app

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/tinywasm/context"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/wizard"
)

// TestWizardFullIntegration is a full integration test that:
// 1. Creates a real project via the wizard
// 2. Creates a real GitHub repository
// 3. Verifies files are in the correct location
// 4. Cleans up by deleting the GitHub repo
//
// Run with: go test -v -run TestWizardFullIntegration .
// Skip with: go test -short .
func TestWizardFullIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Setup: temp parent directory
	parentDir := t.TempDir()
	projectName := "tinywasm-test-" + time.Now().Format("20060102150405")
	projectDir := filepath.Join(parentDir, projectName)

	// Save and restore original working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to parent directory (simulating user running from there)
	if err := os.Chdir(parentDir); err != nil {
		t.Fatal(err)
	}

	// 2. Initialize dependencies (same as Start does)
	gitHandler, err := devflow.NewGit()
	if err != nil {
		t.Fatalf("Failed to create git handler: %v", err)
	}
	gitHandler.SetRootDir(parentDir)

	goHandler, err := devflow.NewGo(gitHandler)
	if err != nil {
		t.Fatalf("Failed to create go handler: %v", err)
	}
	goHandler.SetRootDir(parentDir)

	// GitHub handler for cleanup
	logs := &SafeBuffer{}
	gh, err := devflow.NewGitHub(logs.Log)
	if err != nil {
		t.Fatalf("GitHub unavailable (expected for CI): %v", err)
	}

	ghUser, err := gh.GetCurrentUser()
	if err != nil {
		t.Fatalf("Could not get GitHub user: %v", err)
	}

	// Setup cleanup (delete remote repo after test)
	defer func() {
		if err := gh.DeleteRepo(ghUser, projectName); err != nil {
			t.Logf("Warning: failed to cleanup remote repo: %v", err)
		} else {
			t.Logf("Cleaned up remote repo: %s/%s", ghUser, projectName)
		}
	}()

	// Use a pre-resolved Future to avoid race with SetLog
	githubFuture := devflow.NewResolvedFuture(gh)

	goNew := devflow.NewGoNew(gitHandler, githubFuture, goHandler)
	goNew.SetLog(logs.Log)

	// 3. Create wizard with real GoNew module
	var wizardCompleted bool
	var completedProjectDir string

	var wizardMu sync.Mutex

	w := wizard.New(func(ctx *context.Context) {
		wizardMu.Lock()
		wizardCompleted = true
		completedProjectDir = ctx.Value("project_dir")
		wizardMu.Unlock()
		// Simulate what section-wizard.go does
		if ctx.Value("project_dir") != "" {
			os.Chdir(ctx.Value("project_dir"))
		}
	}, goNew.Module())

	// Provide a mock logger
	w.SetLog(logs.Log)

	// 4. Simulate wizard step inputs
	// Step 1: Project Name
	w.Change(projectName)

	// Step 2: Project Location (use default which is parentDir/projectName)
	w.Change(projectDir)

	// Step 3: Create Project (press Enter)
	w.Change("")

	// 5. Verify wizard completed
	if !wizardCompleted {
		t.Fatal("Wizard did not complete")
	}

	t.Logf("Wizard completed. Project dir: %s", completedProjectDir)

	// 6. Verify files in PROJECT directory (not parent)
	filesToCheck := []string{
		".gitignore",
		"go.mod",
		"README.md",
		"LICENSE",
		projectName + ".go", // handler file
	}

	for _, file := range filesToCheck {
		path := filepath.Join(projectDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s in projectDir, but not found", file)
		}
	}

	// 7. Verify NO files leaked to parent directory
	parentFiles := []string{".gitignore", "go.mod", "README.md"}
	for _, file := range parentFiles {
		path := filepath.Join(parentDir, file)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("File %s should NOT exist in parentDir (leaked)", file)
		}
	}

	t.Logf("Test logs:\n%s", logs.String())
}

// TestWizardLocalOnlyIntegration tests wizard with LocalOnly=true (no GitHub)
func TestWizardLocalOnlyIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	parentDir := t.TempDir()
	projectName := "local-test-project"
	projectDir := filepath.Join(parentDir, projectName)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(parentDir)

	// Create GoNew with nil GitHub (forces LocalOnly)
	gitHandler, _ := devflow.NewGit()
	gitHandler.SetRootDir(parentDir)
	goHandler, _ := devflow.NewGo(gitHandler)

	goNew := devflow.NewGoNew(gitHandler, nil, goHandler)

	logs := &SafeBuffer{}
	goNew.SetLog(logs.Log)

	var wg sync.WaitGroup
	wg.Add(1)

	var wizardCompleted bool

	w := wizard.New(func(ctx *context.Context) {
		wizardCompleted = true
		defer wg.Done()
	}, goNew.Module())

	w.SetLog(logs.Log)

	// Simulate inputs
	w.Change(projectName)
	w.Change(projectDir)
	w.Change("") // Create

	wg.Wait()

	if !wizardCompleted {
		t.Fatal("Wizard did not complete")
	}

	// Verify local files created
	if _, err := os.Stat(filepath.Join(projectDir, ".gitignore")); os.IsNotExist(err) {
		t.Error(".gitignore not found in projectDir")
	}
	if _, err := os.Stat(filepath.Join(projectDir, "go.mod")); os.IsNotExist(err) {
		t.Error("go.mod not found in projectDir")
	}

	t.Logf("Local-only test OK")
}
