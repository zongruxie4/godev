package godev_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/cdvelop/godev"
)

func TestGetScreenSize(t *testing.T) {

	width, height, err := godev.GetScreenSize()

	switch runtime.GOOS {
	case "windows":
		if err != nil {
			t.Errorf("Error inesperado en Windows: %v", err)
		}
		if width <= 0 || height <= 0 {
			t.Errorf("Tamaño del monitor inválido en Windows: %dx%d", width, height)
		}

	case "darwin":
		if err != nil {
			t.Errorf("Error inesperado en macOS: %v", err)
		}
		if width <= 0 || height <= 0 {
			t.Errorf("Tamaño del monitor inválido en macOS: %dx%d", width, height)
		}

	case "linux":
		// En Linux, el archivo virtual_size puede no estar presente o no tener el formato esperado
		// Por lo tanto, no se verifica el error y el tamaño del monitor puede ser cero
		fmt.Printf("Tamaño del monitor en Linux: %dx%d\n", width, height)

	default:
		t.Errorf("Sistema operativo no soportado: %s", runtime.GOOS)
	}
}
