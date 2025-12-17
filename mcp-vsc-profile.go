package app

import (
	"errors"
	"os"
	"path/filepath"
)

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
