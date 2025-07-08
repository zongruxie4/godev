package godev

import "strings"

// GoFileType represents a go file type in backend or frontend domain.
// It allows defining prefixes and specific files for both frontend and backend.
type GoFileType struct {
	FrontendPrefix []string // Prefixes used to identify frontend files (e.g., "f.", "front.")
	FrontendFiles  []string // Specific frontend filenames (e.g., "wasm.main.go")

	BackendPrefix []string // Prefixes used to identify backend files (e.g., "b.", "back.")
	BackendFiles  []string // Specific backend filenames (e.g., "main.server.go")
}

// GoFileIsType checks if a Go file belongs to frontend or backend based on its filename.
// It first checks if the filename starts with any defined prefix (FrontendPrefix or BackendPrefix),
// then checks if the filename matches any specific file (FrontendFiles or BackendFiles).
// Returns two booleans: (isFrontend, isBackend)
// Examples:
//   - "f.index.html" -> (true, false)
//   - "b.main.go" -> (false, true)
//   - "index.html" -> (false, false)
func (ft GoFileType) GoFileIsType(filename string) (bool, bool) {
	if len(filename) < 3 {
		return false, false
	}

	// Check frontend prefixes
	for _, prefix := range ft.FrontendPrefix {
		if strings.HasPrefix(filename, prefix) {
			return true, false
		}
	}

	// Check backend prefixes
	for _, prefix := range ft.BackendPrefix {
		if strings.HasPrefix(filename, prefix) {
			return false, true
		}
	}

	// Check specific frontend files
	for _, file := range ft.FrontendFiles {
		if filename == file {
			return true, false
		}
	}

	// Check specific backend files
	for _, file := range ft.BackendFiles {
		if filename == file {
			return false, true
		}
	}

	return false, false
}
