package godev

// IsFileType checks if a Go file is frontend (f.) or backend (b.)
// Returns (isFrontend bool, isBackend bool)
// Example: "f.index.html" -> true, false
// Example: "b.main.go" -> false, true
// Example: "index.html" -> false, false
func IsFileType(filename string) (bool, bool) {
	if len(filename) < 3 {
		return false, false
	}

	prefix := filename[:2]
	switch prefix {
	case "f.":
		return true, false
	case "b.":
		return false, true
	default:
		return false, false
	}
}
