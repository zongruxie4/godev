package godev

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func findMainFile() (string, error) {
	cmdDir := "cmd"
	mainFiles := []string{}

	// Walk through cmd directory recursively
	err := filepath.Walk(cmdDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is a Go file
		if strings.HasSuffix(info.Name(), ".go") {
			// Read file content
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			// Check if file contains func main() {
			if strings.Contains(string(content), "func main() {") {
				mainFiles = append(mainFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if len(mainFiles) == 0 {
		return "", errors.New("main file not found in cmd directory")
	}

	// If multiple main files found, prefer main.go
	for _, file := range mainFiles {
		if strings.HasSuffix(file, "main.go") {
			return file, nil
		}
	}

	// Return first found main file if main.go not found
	return mainFiles[0], nil
}
