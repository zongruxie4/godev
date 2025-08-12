package godev

import (
	"errors"
	"path/filepath"
	"strings"
)

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
