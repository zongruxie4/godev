package main

import (
	"fmt"
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

	fmt.Println("Nombre aplicación: ", appName)

	return strings.ToLower(appName)
}
