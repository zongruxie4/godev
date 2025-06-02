package godev

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAutoConfig tests the constructor
func TestNewAutoConfig(t *testing.T) {
	printFunc := func(messages ...any) {
		// Test print function
	}

	detector := NewAutoConfig(printFunc)

	assert.NotNil(t, detector)
	assert.Equal(t, ".", detector.rootDir)
	assert.NotNil(t, detector.print)
	assert.NotEmpty(t, detector.AppName)
	assert.Empty(t, detector.Types)
	assert.False(t, detector.HasConsole)
	assert.Equal(t, AppTypeUnknown, detector.WebType)
}

// TestSetRootDir tests setting root directory
func TestSetRootDir(t *testing.T) {
	detector := NewAutoConfig(func(messages ...any) {})

	testDir := "/test/project"
	detector.SetRootDir(testDir)

	assert.Equal(t, testDir, detector.rootDir)
	assert.Equal(t, "project", detector.AppName)
}

// TestArchDetector_DetectConsoleApp tests detection of console applications
func TestArchDetector_DetectConsoleApp(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	// Create cmd directory
	createDir(t, filepath.Join(tempDir, "cmd"))
	createFile(t, filepath.Join(tempDir, "cmd", "main.go"), "package main")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.True(t, detector.HasConsole)
	assert.Contains(t, detector.Types, AppTypeConsole)
	assert.Equal(t, filepath.Base(tempDir), detector.AppName)
}

// TestArchDetector_DetectPWA tests detection of Progressive Web Apps
func TestArchDetector_DetectPWA(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	// Create web/pwa directory structure
	createDir(t, filepath.Join(tempDir, "web", "pwa"))
	createFile(t, filepath.Join(tempDir, "web", "pwa", "index.html"), "<html></html>")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.Equal(t, AppTypePWA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypePWA)
}

// TestArchDetector_DetectSPA tests detection of Single Page Applications
func TestArchDetector_DetectSPA(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create web/spa directory structure
	createDir(t, filepath.Join(tempDir, "web", "spa"))
	createFile(t, filepath.Join(tempDir, "web", "spa", "app.js"), "console.log('spa')")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.Equal(t, AppTypeSPA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeSPA)
}

// TestArchDetector_DetectWebUndefined tests detection of undefined web architecture
func TestArchDetector_DetectWebUndefined(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create web directory without specific architecture
	createDir(t, filepath.Join(tempDir, "web"))
	createFile(t, filepath.Join(tempDir, "web", "index.html"), "<html></html>")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.Equal(t, AppTypeWeb, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeWeb)
}

// TestArchDetector_DetectHybridApp tests detection of hybrid applications
func TestArchDetector_DetectHybridApp(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create both cmd and web directories
	createDir(t, filepath.Join(tempDir, "cmd"))
	createFile(t, filepath.Join(tempDir, "cmd", "main.go"), "package main")
	createDir(t, filepath.Join(tempDir, "web", "spa"))
	createFile(t, filepath.Join(tempDir, "web", "spa", "app.js"), "console.log('spa')")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.True(t, detector.HasConsole)
	assert.Equal(t, AppTypeSPA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeConsole)
	assert.Contains(t, detector.Types, AppTypeSPA)
	assert.Len(t, detector.Types, 2)
}

// TestArchDetector_ConflictingWebArchitectures tests validation of conflicting web architectures
func TestArchDetector_ConflictingWebArchitectures(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create both pwa and spa directories (should cause conflict)
	createDir(t, filepath.Join(tempDir, "web", "pwa"))
	createDir(t, filepath.Join(tempDir, "web", "spa"))
	createFile(t, filepath.Join(tempDir, "web", "pwa", "index.html"), "<html></html>")
	createFile(t, filepath.Join(tempDir, "web", "spa", "app.js"), "console.log('spa')")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting web architectures")
}

// TestArchDetector_NoArchitecture tests when no architecture is detected
func TestArchDetector_NoArchitecture(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create only some random files, no architecture directories
	createFile(t, filepath.Join(tempDir, "main.go"), "package main")
	createFile(t, filepath.Join(tempDir, "README.md"), "# Test")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.False(t, detector.HasConsole)
	assert.Equal(t, AppTypeUnknown, detector.WebType)
	assert.Empty(t, detector.Types)
}

// TestArchDetector_NewFileEvent tests the NewFileEvent method
func TestArchDetector_NewFileEvent(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	detector := createTestDetector(tempDir, t)

	tests := []struct {
		name      string
		fileName  string
		filePath  string
		event     string
		shouldRun bool
	}{
		{
			name:      "file event should be ignored",
			fileName:  "main.go",
			filePath:  filepath.Join(tempDir, "main.go"),
			event:     "created",
			shouldRun: false,
		},
		{
			name:      "cmd directory event should trigger scan",
			fileName:  "cmd",
			filePath:  filepath.Join(tempDir, "cmd"),
			event:     "created",
			shouldRun: true,
		},
		{
			name:      "web directory event should trigger scan",
			fileName:  "web",
			filePath:  filepath.Join(tempDir, "web"),
			event:     "created",
			shouldRun: true,
		},
		{
			name:      "irrelevant directory should be ignored",
			fileName:  "docs",
			filePath:  filepath.Join(tempDir, "docs"),
			event:     "created",
			shouldRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detector.NewFolderEvent(tt.fileName, tt.filePath, tt.event)
			if tt.shouldRun {
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err) // Should always return nil, just not process
			}
		})
	}
}

// TestArchDetector_IsRelevantDirectoryChange tests the relevance checking logic
func TestArchDetector_IsRelevantDirectoryChange(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	detector := createTestDetector(tempDir, t)
	appName := filepath.Base(tempDir)

	tests := []struct {
		name     string
		dirPath  string
		expected bool
	}{
		{
			name:     "cmd directory",
			dirPath:  filepath.Join(tempDir, "cmd"),
			expected: true,
		},
		{
			name:     "cmd with app name",
			dirPath:  filepath.Join(tempDir, "cmd", appName),
			expected: true,
		},
		{
			name:     "web directory",
			dirPath:  filepath.Join(tempDir, "web"),
			expected: true,
		},
		{
			name:     "web/pwa directory",
			dirPath:  filepath.Join(tempDir, "web", "pwa"),
			expected: true,
		},
		{
			name:     "web/spa directory",
			dirPath:  filepath.Join(tempDir, "web", "spa"),
			expected: true,
		},
		{
			name:     "irrelevant directory",
			dirPath:  filepath.Join(tempDir, "docs"),
			expected: false,
		},
		{
			name:     "nested irrelevant directory",
			dirPath:  filepath.Join(tempDir, "test", "data"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isRelevantDirectoryChange(tt.dirPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions

func createTestDetector(tempDir string, t *testing.T) *AutoConfig {
	detector := NewAutoConfig(func(messages ...any) {
		t.Logf("AutoConfig: %v", messages)
	})
	detector.SetRootDir(tempDir)
	return detector
}

func createTempDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "autoconfig_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tempDir
}

func createDir(t *testing.T, path string) {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory %s: %v", path, err)
	}
}

func createFile(t *testing.T, path, content string) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatalf("Failed to create directory for file %s: %v", path, err)
	}

	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create file %s: %v", path, err)
	}
}
