package app

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// ConfigureVSCodeMCP attempts to automatically configure VS Code's MCP integration.
// This function is completely silent and non-blocking - it will not produce errors or logs.
// It detects VS Code installation, resolves all profiles, and updates mcp.json files
// with TinyWasm's MCP server configuration.
//
// Behavior:
//   - Detects platform-specific VS Code paths (Linux, macOS, Windows)
//   - Updates ALL profiles (handles multiple profiles by updating all)
//   - Creates or updates mcp.json with tinywasm-mcp entry in each profile
//   - Fails silently on any error (VS Code not found, permissions, etc.)
//
// Inspired by TinyWasm's VisualStudioCodeWasmEnvConfig approach.
func ConfigureVSCodeMCP() {
	// Get platform-specific VS Code path
	basePath, err := getVSCodeConfigPath()
	if err != nil {
		return // Silent failure: unsupported platform or missing home dir
	}

	// Resolve all profile paths (or base path)
	configPaths, err := findMCPConfigPaths(basePath)
	if err != nil {
		return // Silent failure: VS Code not installed or no profiles
	}

	// Update configuration in all profiles
	for _, configPath := range configPaths {
		_ = updateMCPConfig(configPath, MCPPort)
	}
}

// getVSCodeConfigPath returns the platform-specific VS Code User directory path.
// Returns error if platform is unsupported or environment variables are missing.
func getVSCodeConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "linux":
		return filepath.Join(homeDir, ".config", "Code", "User"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Code", "User"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", errors.New("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "Code", "User"), nil
	default:
		return "", errors.New("unsupported platform: " + runtime.GOOS)
	}
}

// findMCPConfigPaths resolves all mcp.json file paths based on VS Code profile structure.
// Logic:
//   - If profiles/ directory doesn't exist: return base path
//   - If profiles exist: return all profile paths (will update all)
func findMCPConfigPaths(basePath string) ([]string, error) {
	// Check if VS Code User directory exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil, errors.New("VS Code not installed")
	}

	profilesPath := filepath.Join(basePath, "profiles")

	// Check if profiles directory exists
	if _, err := os.Stat(profilesPath); os.IsNotExist(err) {
		// No profiles, use base path
		return []string{filepath.Join(basePath, "mcp.json")}, nil
	}

	// Get all profile directories
	entries, err := os.ReadDir(profilesPath)
	if err != nil {
		return nil, err
	}

	configPaths := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			configPaths = append(configPaths, filepath.Join(profilesPath, entry.Name(), "mcp.json"))
		}
	}

	if len(configPaths) == 0 {
		return nil, errors.New("no profiles found")
	}

	return configPaths, nil
}
