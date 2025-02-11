package godev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateFileOnDisk(t *testing.T) {
	// Configurar entorno de prueba
	testDir := "test"
	buildDir := filepath.Join(testDir, "build")
	styleCssPath := filepath.Join(buildDir, "style.css")
	defer os.Remove(styleCssPath)

	// Crear directorio build si no existe
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("Error creando directorio de build: %v", err)
	}

	// Configuración mock del compilador
	config := &AssetsConfig{
		BuildDirectory: func() string { return buildDir },
		Print: func(messages ...any) {
			fmt.Println(messages...)
		},
		WasmProjectTinyGoJsUse: func() (bool, bool) {
			return false, false
		},
	}
	assetsCompiler := NewAssetsCompiler(config)

	t.Run("Crear nuevo archivo CSS", func(t *testing.T) {
		cssPath := filepath.Join(testDir, "test.css")
		defer os.Remove(cssPath)

		// Crear archivo CSS de prueba
		if err := os.WriteFile(cssPath, []byte(".test { color: red; }"), 0644); err != nil {
			t.Fatal(err)
		}

		// Ejecutar función bajo prueba
		if err := assetsCompiler.UpdateFileOnDisk(cssPath, ".css"); err != nil {
			t.Fatalf("Error inesperado: %v", err)
		}

		// Verificar archivo generado
		if _, err := os.Stat(styleCssPath); os.IsNotExist(err) {
			t.Fatal("Archivo CSS no generado")
		}

		// Verificar contenido minificado
		content, _ := os.ReadFile(styleCssPath)
		if string(content) != ".test{color:red}" {
			t.Fatalf("Contenido CSS minificado incorrecto: %s", content)
		}
	})

	t.Run("Actualizar archivo CSS existente", func(t *testing.T) {
		cssPath := filepath.Join(testDir, "existing.css")
		defer os.Remove(cssPath)

		// Crear archivo inicial
		os.WriteFile(cssPath, []byte(".old { padding: 1px; }"), 0644)
		assetsCompiler.UpdateFileOnDisk(cssPath, ".css")

		// Actualizar contenido
		os.WriteFile(cssPath, []byte(".new { margin: 2px; }"), 0644)
		if err := assetsCompiler.UpdateFileOnDisk(cssPath, ".css"); err != nil {
			t.Fatal(err)
		}
		expected := ".new{margin:2px}"

		// Verificar actualización
		gotByte, _ := os.ReadFile(styleCssPath)
		got := string(gotByte)

		if !strings.Contains(got, expected) {
			t.Fatalf("\nexpected not found: \n%s\ngot: \n%s\n", expected, got)
		}

	})

	t.Run("Manejar archivo inexistente", func(t *testing.T) {
		err := assetsCompiler.UpdateFileOnDisk("no_existe.css", ".css")
		if err == nil {
			t.Fatal("Se esperaba error por archivo no encontrado")
		}
	})

	t.Run("Extensión inválida", func(t *testing.T) {
		err := assetsCompiler.UpdateFileOnDisk("archivo.txt", ".txt")
		if err == nil {
			t.Fatal("Se esperaba error por extensión inválida")
		}
	})

	t.Run("Crear archivo JS básico", func(t *testing.T) {
		jsPath := filepath.Join(testDir, "test.js")
		defer os.Remove(jsPath)

		os.WriteFile(jsPath, []byte(`// Test\nfunction hello() { console.log("hola") }
		let x = 10;`), 0644)

		if err := assetsCompiler.UpdateFileOnDisk(jsPath, ".js"); err != nil {
			t.Fatal(err)
		}

		got, _ := os.ReadFile(filepath.Join(buildDir, "main.js"))
		expected := "'use strict';let x=10"
		if string(got) != expected {
			t.Fatalf("\nJS minificado incorrecto:\nexpected: \n[%s]\n\ngot: \n[%s]", expected, got)
		}
	})
}
