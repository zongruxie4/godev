package test

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func findFilesWithNonZeroSize(dir string, filenames []string) error {

	// Esperar
	time.Sleep(200 * time.Millisecond)

	// Crea un mapa para hacer un seguimiento de los archivos encontrados
	found := make(map[string]bool)
	for _, filename := range filenames {
		found[filename] = false
	}

	// Recorre el directorio en busca de archivos
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Comprueba si el archivo actual es uno de los que estamos buscando
		filename := filepath.Base(path)
		if _, ok := found[filename]; ok && info.Size() > 0 {
			found[filename] = true
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Verifica que se encontraron todos los archivos y que tienen tamaño mayor que cero
	for filename, ok := range found {
		if !ok {
			return fmt.Errorf("no se encontró el archivo %s", filename)
		}

	}

	return nil
}
