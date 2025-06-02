package godev

import (
	"errors"
	"path/filepath"
	"strings"
)

// GetFileName returns the filename from a path
// Example: "theme/index.html" -> "index.html"
func GetFileName(path string) (string, error) {
	if path == "" {
		return "", errors.New("GetFileName empty path")
	}

	// Check if path ends with a separator
	if len(path) > 0 && (path[len(path)-1] == '/' || path[len(path)-1] == '\\') {
		return "", errors.New("GetFileName invalid path: ends with separator")
	}

	fileName := filepath.Base(path)
	if fileName == "." || fileName == string(filepath.Separator) {
		return "", errors.New("GetFileName invalid path")
	}
	if len(path) > 0 && path[len(path)-1] == filepath.Separator {
		return "", errors.New("GetFileName invalid path")
	}

	return fileName, nil
}

// GetModuleName extracts the module name from a path containing a "modules" directory
// Example: "project/modules/user/model.go" -> "user"
func GetModuleName(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}

	// Normalize path separators and clean the path
	cleanPath := filepath.Clean(path)

	// Split into parts using OS-specific separator
	parts := strings.Split(cleanPath, string(filepath.Separator))

	// Find the "modules" directory and return the next part
	for i, part := range parts {
		if part == "modules" {
			if i+1 >= len(parts) {
				return "", errors.New("path ends at modules directory")
			}

			nextPart := parts[i+1]
			if nextPart == "" || nextPart == "." || nextPart == ".." {
				return "", errors.New("invalid module name")
			}

			return nextPart, nil
		}
	}

	return "", errors.New("modules directory not found")
}
