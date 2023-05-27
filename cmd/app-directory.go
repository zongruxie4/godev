package main

import (
	"log"
	"os"
	"path/filepath"
)

func getAppDirectory() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Error al obtener la ruta del directorio de la aplicación:", err)
	}

	appDir := filepath.Dir(exePath)
	return appDir
}
