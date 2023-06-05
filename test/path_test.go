package test

import (
	"os"
	"testing"

	"github.com/cdvelop/theme_platform"
)

func Test_Path(t *testing.T) {

	themePath := theme_platform.Theme{}.FolderPath()

	// fmt.Println("DIRECTORIO TEMA: ", themePath)
	// Verificar si la ruta existe
	_, err := os.Stat(themePath)
	if err != nil {
		t.Errorf("La ruta del tema no existe: %s", themePath)
	}

	// Verificar si la ruta es un directorio
	fileInfo, err := os.Stat(themePath)
	if err != nil || !fileInfo.IsDir() {
		t.Errorf("La ruta del tema no es un directorio válido: %s", themePath)
	}

}
