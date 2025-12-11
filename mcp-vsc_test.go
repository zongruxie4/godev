package tinywasm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestGetVSCodeConfigPath verifies platform-specific path detection
func TestGetVSCodeConfigPath(t *testing.T) {
	path, err := getVSCodeConfigPath()
	if err != nil {
		t.Fatalf("getVSCodeConfigPath() failed: %v", err)
	}

	if path == "" {
		t.Fatal("getVSCodeConfigPath() returned empty path")
	}

	// Verify platform-specific suffix
	switch runtime.GOOS {
	case "linux":
		if !filepath.IsAbs(path) || !contains(path, ".config/Code/User") {
			t.Errorf("Linux path incorrect: %s", path)
		}
	case "darwin":
		if !filepath.IsAbs(path) || !contains(path, "Library/Application Support/Code/User") {
			t.Errorf("macOS path incorrect: %s", path)
		}
	case "windows":
		if !filepath.IsAbs(path) || !contains(path, "Code\\User") {
			t.Errorf("Windows path incorrect: %s", path)
		}
	}
}

// TestFindMCPConfigPaths tests profile resolution logic
func TestFindMCPConfigPaths(t *testing.T) {
	// Create temporary test structure
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setup       func() string
		expectError bool
		expectCount int
		expectPaths []string
	}{
		{
			name: "no profiles directory",
			setup: func() string {
				return tempDir
			},
			expectError: false,
			expectCount: 1,
			expectPaths: []string{filepath.Join(tempDir, "mcp.json")},
		},
		{
			name: "single profile",
			setup: func() string {
				profileDir := filepath.Join(tempDir, "profiles", "test-profile")
				os.MkdirAll(profileDir, 0755)
				return tempDir
			},
			expectError: false,
			expectCount: 1,
			expectPaths: []string{filepath.Join(tempDir, "profiles", "test-profile", "mcp.json")},
		},
		{
			name: "multiple profiles",
			setup: func() string {
				os.MkdirAll(filepath.Join(tempDir, "profiles", "profile1"), 0755)
				os.MkdirAll(filepath.Join(tempDir, "profiles", "profile2"), 0755)
				return tempDir
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "non-existent base path",
			setup: func() string {
				return filepath.Join(tempDir, "nonexistent")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean temp dir for each test
			os.RemoveAll(filepath.Join(tempDir, "profiles"))

			basePath := tt.setup()
			paths, err := findMCPConfigPaths(basePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(paths) != tt.expectCount {
					t.Errorf("Expected %d paths, got %d", tt.expectCount, len(paths))
				}
				if tt.expectPaths != nil {
					for i, expectedPath := range tt.expectPaths {
						if i < len(paths) && paths[i] != expectedPath {
							t.Errorf("Expected path[%d] %s, got %s", i, expectedPath, paths[i])
						}
					}
				}
			}
		})
	}
}

// TestUpdateMCPConfig tests JSON manipulation logic
func TestUpdateMCPConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "mcp.json")

	// Test 1: Create new config
	err := updateMCPConfig(configPath, "3030")
	if err != nil {
		t.Fatalf("Failed to create new config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Verify content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var config mcpConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if config.Servers == nil {
		t.Fatal("Servers map is nil")
	}

	goliteConfig, exists := config.Servers["golite-mcp"]
	if !exists {
		t.Fatal("golite-mcp entry not found")
	}

	if goliteConfig.URL != "http://localhost:3030/mcp" {
		t.Errorf("Expected URL http://localhost:3030/mcp, got %s", goliteConfig.URL)
	}

	if goliteConfig.Type != "http" {
		t.Errorf("Expected type http, got %s", goliteConfig.Type)
	}

	// Test 2: Update existing config
	err = updateMCPConfig(configPath, "8080")
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Verify update
	data, _ = os.ReadFile(configPath)
	json.Unmarshal(data, &config)
	goliteConfig = config.Servers["golite-mcp"]

	if goliteConfig.URL != "http://localhost:8080/mcp" {
		t.Errorf("URL not updated: %s", goliteConfig.URL)
	}

	// Test 3: Preserve other servers
	config.Servers["other-server"] = mcpServerConfig{
		Type:    "stdio",
		Command: "test",
	}
	data, _ = json.MarshalIndent(config, "", "\t")
	os.WriteFile(configPath, data, 0644)

	err = updateMCPConfig(configPath, "3030")
	if err != nil {
		t.Fatalf("Failed to update config with existing servers: %v", err)
	}

	data, _ = os.ReadFile(configPath)
	json.Unmarshal(data, &config)

	if len(config.Servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(config.Servers))
	}

	if _, exists := config.Servers["other-server"]; !exists {
		t.Error("Other server was removed")
	}
}

// TestConfigureVSCodeMCP tests the public API (basic smoke test)
func TestConfigureVSCodeMCP(t *testing.T) {
	// This should not panic or block, even if VS Code isn't installed
	ConfigureVSCodeMCP()
	// Success means it didn't panic
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			filepath.Base(filepath.Dir(s)) == filepath.Base(filepath.Dir(substr))))
}
