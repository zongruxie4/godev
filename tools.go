package godev

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// pathFile ej: "theme/index.html"
// data ej: *bytes.Buffer
// NOTA: la data del buf sera eliminada después de escribir el archivo
func FileWrite(pathFile string, data bytes.Buffer) error {
	const e = "FileWrite "
	dst, err := os.Create(pathFile)
	if err != nil {
		if strings.Contains(err.Error(), "system cannot find the path") {
			dir := filepath.Dir(pathFile)
			os.MkdirAll(dir, 0777)
			dst, err = os.Create(pathFile)
			if err != nil {
				return errors.New(e + "al crear archivo " + err.Error())
			}
		} else {
			return errors.New(e + "al crear archivo " + err.Error())
		}

	}
	defer dst.Close()

	// fmt.Println("data antes de escribir:", data.String())
	// Copy the uploaded File to the filesystem at the specified destination
	// _, e = io.Copy(dst, bytes.NewReader(data.Bytes()))
	_, err = io.Copy(dst, &data)
	if err != nil {
		return errors.New(e + "no se logro escribir el archivo " + pathFile + " en el destino " + err.Error())
	}
	// fmt.Println("data después de copy:", data.String(), "bytes:", data.Bytes())

	return nil
}

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
