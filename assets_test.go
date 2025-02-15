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
		WebFilesFolder: func() string { return buildDir },
		Print: func(messages ...any) {
			fmt.Println(messages...)
		},
		WasmProjectTinyGoJsUse: func() (bool, bool) {
			return false, false
		},
	}
	assetsHandler := NewAssetsCompiler(config)

	t.Run("Crear nuevo archivo CSS", func(t *testing.T) {
		cssPath := filepath.Join(testDir, "test.css")

		// Crear archivo CSS de prueba
		if err := os.WriteFile(cssPath, []byte(".test { color: red; }"), 0644); err != nil {
			t.Fatal(err)
		}

		// Ejecutar función bajo prueba
		if err := assetsHandler.UpdateFileOnDisk(cssPath, ".css"); err != nil {
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

		// Crear archivo inicial
		os.WriteFile(cssPath, []byte(".old { padding: 1px; }"), 0644)
		assetsHandler.UpdateFileOnDisk(cssPath, ".css")

		// Actualizar contenido
		os.WriteFile(cssPath, []byte(".new { margin: 2px; }"), 0644)
		if err := assetsHandler.UpdateFileOnDisk(cssPath, ".css"); err != nil {
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
		err := assetsHandler.UpdateFileOnDisk("no_existe.css", ".css")
		if err == nil {
			t.Fatal("Se esperaba error por archivo no encontrado")
		}
	})

	t.Run("Extensión inválida", func(t *testing.T) {
		err := assetsHandler.UpdateFileOnDisk("archivo.txt", ".txt")
		if err == nil {
			t.Fatal("Se esperaba error por extensión inválida")
		}
	})

	t.Run("Crear archivo JS básico", func(t *testing.T) {
		jsPath := filepath.Join(testDir, "test.js")
		jsPath2 := filepath.Join(testDir, "test2.js")
		defer os.Remove(jsPath)
		defer os.Remove(jsPath2)

		os.WriteFile(jsPath, []byte(`// Test\nfunction hello() { console.log("hola") }
		let x = 10;`), 0644)
		os.WriteFile(jsPath2, []byte(`// Test2\nfunction bye() { console.log("adios") }
		let y = 20;`), 0644)

		if err := assetsHandler.UpdateFileOnDisk(jsPath, ".js"); err != nil {
			t.Fatal(err)
		}
		if err := assetsHandler.UpdateFileOnDisk(jsPath2, ".js"); err != nil {
			t.Fatal(err)
		}

		got, _ := os.ReadFile(filepath.Join(buildDir, "main.js"))
		expected := `"use strict";let x=10,y=20`
		if string(got) != expected {
			t.Fatalf("\nJS minificado incorrecto:\nexpected: \n[%s]\n\ngot: \n[%s]", expected, got)
		}

	})
}
