package godev

import (
	"io"
	"os"
	"path/filepath"
)

func (u ui) copyStaticFilesFromUiTheme() {
	// Definir las extensiones o tipos de archivo permitidos
	validExtensions := map[string]bool{".js": true, ".css": true, ".wasm": true}

	// Obtener la lista de archivos en la carpeta origen
	srcDir := u.FolderPath() + "/static"
	destDir := BuiltFolder + "/static"
	files, err := os.ReadDir(srcDir)
	if err != nil {
		panic(err)
	}

	// Recorrer la lista de archivos
	for _, file := range files {
		// Verificar si el archivo no es de una extensión prohibida
		ext := filepath.Ext(file.Name())
		if !validExtensions[ext] {
			// Obtener la ruta completa del archivo origen y destino
			src := filepath.Join(srcDir, file.Name())
			dest := filepath.Join(destDir, file.Name())

			// Verificar si el archivo destino ya existe
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				// Si el archivo destino no existe, copiar el archivo
				err := copyFile(src, dest)
				if err != nil {
					panic(err)
				}
			}
		}
	}
}

func copyFile(src string, dest string) error {
	// Abrir el archivo origen
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Crear el archivo destino
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copiar el contenido del archivo origen al archivo destino
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}
