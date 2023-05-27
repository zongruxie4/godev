package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func getAppName() string {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error al obtener el directorio actual:", err)
	}

	// Obtiene el nombre de la carpeta actual como nombre de la aplicación
	_, appName := filepath.Split(currentDir)
	return strings.ToLower(appName)
}
