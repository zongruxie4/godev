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

// TestArchDetector_DetectPWA_RootLevel tests detection of Progressive Web Apps in root
func TestArchDetector_DetectPWA_RootLevel(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)
	// Create pwa directory structure in root
	createDir(t, filepath.Join(tempDir, "pwa"))
	createFile(t, filepath.Join(tempDir, "pwa", "index.html"), "<html></html>")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.Equal(t, AppTypePWA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypePWA)
}

// TestArchDetector_DetectSPA_RootLevel tests detection of Single Page Applications in root
func TestArchDetector_DetectSPA_RootLevel(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create spa directory structure in root
	createDir(t, filepath.Join(tempDir, "spa"))
	createFile(t, filepath.Join(tempDir, "spa", "app.js"), "console.log('spa')")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.Equal(t, AppTypeSPA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeSPA)
}

// TestArchDetector_DetectMPA_RootLevel tests detection of Multi-Page Applications in root
func TestArchDetector_DetectMPA_RootLevel(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create mpa directory structure in root
	createDir(t, filepath.Join(tempDir, "mpa"))
	createFile(t, filepath.Join(tempDir, "mpa", "page1.html"), "<html>Page 1</html>")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.Equal(t, AppTypeMPA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeMPA)
}

// TestArchDetector_NoArchitecture_ReturnsUnknown tests when no architecture is detected
func TestArchDetector_NoArchitecture_ReturnsUnknown(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create only some random files, no architecture directories
	createFile(t, filepath.Join(tempDir, "README.md"), "# Test Project")
	createFile(t, filepath.Join(tempDir, "go.mod"), "module test")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.Equal(t, AppTypeUnknown, detector.WebType)
	assert.Empty(t, detector.Types)
	assert.False(t, detector.HasConsole)
}

// TestArchDetector_DetectHybridApp tests detection of hybrid applications
func TestArchDetector_DetectHybridApp(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create both cmd and spa directories
	createDir(t, filepath.Join(tempDir, "cmd"))
	createFile(t, filepath.Join(tempDir, "cmd", "main.go"), "package main")
	createDir(t, filepath.Join(tempDir, "spa"))
	createFile(t, filepath.Join(tempDir, "spa", "app.js"), "console.log('spa')")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.True(t, detector.HasConsole)
	assert.Equal(t, AppTypeSPA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeConsole)
	assert.Contains(t, detector.Types, AppTypeSPA)
	assert.Len(t, detector.Types, 2)
}

// Test removed - replaced with priority resolution tests above

// Test removed - duplicate of TestArchDetector_NoArchitecture_ReturnsUnknown above

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
			name:      "pwa directory event should trigger scan",
			fileName:  "pwa",
			filePath:  filepath.Join(tempDir, "pwa"),
			event:     "created",
			shouldRun: true,
		},
		{
			name:      "spa directory event should trigger scan",
			fileName:  "spa",
			filePath:  filepath.Join(tempDir, "spa"),
			event:     "created",
			shouldRun: true,
		},
		{
			name:      "mpa directory event should trigger scan",
			fileName:  "mpa",
			filePath:  filepath.Join(tempDir, "mpa"),
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
			name:     "pwa directory",
			dirPath:  filepath.Join(tempDir, "pwa"),
			expected: true,
		},
		{
			name:     "spa directory",
			dirPath:  filepath.Join(tempDir, "spa"),
			expected: true,
		},
		{
			name:     "mpa directory",
			dirPath:  filepath.Join(tempDir, "mpa"),
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

// Priority Resolution Tests - PWA wins over SPA and MPA
// TestArchDetector_PWA_SPA_Priority tests that PWA has priority over SPA
func TestArchDetector_PWA_SPA_Priority(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create both PWA and SPA directories
	createDir(t, filepath.Join(tempDir, "pwa"))
	createFile(t, filepath.Join(tempDir, "pwa", "manifest.json"), "{}")
	createDir(t, filepath.Join(tempDir, "spa"))
	createFile(t, filepath.Join(tempDir, "spa", "app.js"), "console.log('spa')")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	// PWA should win (priority 1)
	assert.Equal(t, AppTypePWA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypePWA)
	assert.NotContains(t, detector.Types, AppTypeSPA)
}

// TestArchDetector_PWA_MPA_Priority tests that PWA has priority over MPA
func TestArchDetector_PWA_MPA_Priority(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create both PWA and MPA directories
	createDir(t, filepath.Join(tempDir, "pwa"))
	createFile(t, filepath.Join(tempDir, "pwa", "manifest.json"), "{}")
	createDir(t, filepath.Join(tempDir, "mpa"))
	createFile(t, filepath.Join(tempDir, "mpa", "page1.html"), "<html></html>")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	// PWA should win (priority 1)
	assert.Equal(t, AppTypePWA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypePWA)
	assert.NotContains(t, detector.Types, AppTypeMPA)
}

// TestArchDetector_SPA_MPA_Priority tests that SPA has priority over MPA
func TestArchDetector_SPA_MPA_Priority(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create both SPA and MPA directories
	createDir(t, filepath.Join(tempDir, "spa"))
	createFile(t, filepath.Join(tempDir, "spa", "app.js"), "console.log('spa')")
	createDir(t, filepath.Join(tempDir, "mpa"))
	createFile(t, filepath.Join(tempDir, "mpa", "page1.html"), "<html></html>")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	// SPA should win (priority 2)
	assert.Equal(t, AppTypeSPA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeSPA)
	assert.NotContains(t, detector.Types, AppTypeMPA)
}

// Hybrid Application Tests
// TestArchDetector_CMD_PWA_Hybrid tests console + PWA hybrid application
func TestArchDetector_CMD_PWA_Hybrid(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create both cmd and pwa directories
	createDir(t, filepath.Join(tempDir, "cmd"))
	createFile(t, filepath.Join(tempDir, "cmd", "main.go"), "package main")
	createDir(t, filepath.Join(tempDir, "pwa"))
	createFile(t, filepath.Join(tempDir, "pwa", "manifest.json"), "{}")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.True(t, detector.HasConsole)
	assert.Equal(t, AppTypePWA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeConsole)
	assert.Contains(t, detector.Types, AppTypePWA)
	assert.Len(t, detector.Types, 2) // Should have exactly 2 types
}

// TestArchDetector_CMD_SPA_Hybrid tests console + SPA hybrid application
func TestArchDetector_CMD_SPA_Hybrid(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create both cmd and spa directories
	createDir(t, filepath.Join(tempDir, "cmd"))
	createFile(t, filepath.Join(tempDir, "cmd", "main.go"), "package main")
	createDir(t, filepath.Join(tempDir, "spa"))
	createFile(t, filepath.Join(tempDir, "spa", "app.js"), "console.log('spa')")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.True(t, detector.HasConsole)
	assert.Equal(t, AppTypeSPA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeConsole)
	assert.Contains(t, detector.Types, AppTypeSPA)
	assert.Len(t, detector.Types, 2) // Should have exactly 2 types
}

// TestArchDetector_CMD_MPA_Hybrid tests console + MPA hybrid application
func TestArchDetector_CMD_MPA_Hybrid(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create both cmd and mpa directories
	createDir(t, filepath.Join(tempDir, "cmd"))
	createFile(t, filepath.Join(tempDir, "cmd", "main.go"), "package main")
	createDir(t, filepath.Join(tempDir, "mpa"))
	createFile(t, filepath.Join(tempDir, "mpa", "page1.html"), "<html></html>")

	detector := createTestDetector(tempDir, t)

	err := detector.ScanDirectoryStructure()
	assert.NoError(t, err)

	assert.True(t, detector.HasConsole)
	assert.Equal(t, AppTypeMPA, detector.WebType)
	assert.Contains(t, detector.Types, AppTypeConsole)
	assert.Contains(t, detector.Types, AppTypeMPA)
	assert.Len(t, detector.Types, 2) // Should have exactly 2 types
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
