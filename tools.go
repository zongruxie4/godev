package godev

import (
	"bytes"
	"errors"
	"io"
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
func GetFileName(path string) (string, error) {
	if path == "" {
		return "", errors.New("GetFileName empty path")
	}

	fileName := filepath.Base(path)
	if fileName == "." || fileName == string(filepath.Separator) {
		return "", errors.New("GetFileName invalid path")
	}

	return fileName, nil
}
