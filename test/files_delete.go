package test

import (
	"fmt"
	"os"
	"path/filepath"
)

func deleteFiles(dir string, exts []string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("Error reading directory: %s\n", err)
		return
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		for _, ext := range exts {
			if ext == filepath.Ext(file.Name()) {
				path := filepath.Join(dir, file.Name())
				err := os.Remove(path)
				if err != nil {
					fmt.Printf("Error deleting file: %s\n", err)
				}
				break
			}
		}
	}
}
