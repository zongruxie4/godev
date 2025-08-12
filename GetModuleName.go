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

	// Replace backslashes with slashes to support Windows paths on any OS
	normalized := strings.ReplaceAll(path, "\\", "/")

	// Clean the path
	cleanPath := filepath.Clean(normalized)

	// Split into parts using '/' to be OS-agnostic
	parts := strings.Split(cleanPath, "/")

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
