package godev

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateFileOnDisk(t *testing.T) {
	// Configurar entorno de prueba
	testDir := "test"
	buildDir := filepath.Join(testDir, "build")

	// Crear directorio build si no existe
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("Error creando directorio de build: %v", err)
	}

	// Configuración mock del compilador
	config := &CompilerConfig{
		BuildDirectory: func() string { return buildDir },
		Println:        func(...any) {},
		WasmProjectTinyGoJsUse: func() (bool, bool) {
			return false, false
		},
	}
	compiler := New(config)

	t.Run("Crear nuevo archivo CSS", func(t *testing.T) {
		cssPath := filepath.Join(testDir, "test.css")
		defer os.Remove(cssPath)

		// Crear archivo CSS de prueba
		if err := os.WriteFile(cssPath, []byte(".test { color: red; }"), 0644); err != nil {
			t.Fatal(err)
		}

		// Ejecutar función bajo prueba
		if err := compiler.UpdateFileOnDisk(cssPath, ".css"); err != nil {
			t.Fatalf("Error inesperado: %v", err)
		}

		// Verificar archivo generado
		generated := filepath.Join(buildDir, "style.css")
		if _, err := os.Stat(generated); os.IsNotExist(err) {
			t.Fatal("Archivo CSS no generado")
		}

		// Verificar contenido minificado
		content, _ := os.ReadFile(generated)
		if string(content) != ".test{color:red}" {
			t.Errorf("Contenido CSS minificado incorrecto: %s", content)
		}
	})

	t.Run("Actualizar archivo CSS existente", func(t *testing.T) {
		cssPath := filepath.Join(testDir, "existing.css")
		defer os.Remove(cssPath)

		// Crear archivo inicial
		os.WriteFile(cssPath, []byte(".old { padding: 0; }"), 0644)
		compiler.UpdateFileOnDisk(cssPath, ".css")

		// Actualizar contenido
		os.WriteFile(cssPath, []byte(".new { margin: 0; }"), 0644)
		if err := compiler.UpdateFileOnDisk(cssPath, ".css"); err != nil {
			t.Fatal(err)
		}

		// Verificar actualización
		content, _ := os.ReadFile(filepath.Join(buildDir, "style.css"))
		if string(content) != ".new{margin:0}" {
			t.Error("CSS no se actualizó correctamente")
		}
	})

	t.Run("Manejar archivo inexistente", func(t *testing.T) {
		err := compiler.UpdateFileOnDisk("no_existe.css", ".css")
		if err == nil {
			t.Error("Se esperaba error por archivo no encontrado")
		}
	})

	t.Run("Extensión inválida", func(t *testing.T) {
		err := compiler.UpdateFileOnDisk("archivo.txt", ".txt")
		if err == nil {
			t.Error("Se esperaba error por extensión inválida")
		}
	})

	t.Run("Crear archivo JS básico", func(t *testing.T) {
		jsPath := filepath.Join(testDir, "test.js")
		defer os.Remove(jsPath)

		os.WriteFile(jsPath, []byte("// Test\nfunction hello() { console.log('hola') }"), 0644)

		if err := compiler.UpdateFileOnDisk(jsPath, ".js"); err != nil {
			t.Fatal(err)
		}

		content, _ := os.ReadFile(filepath.Join(buildDir, "main.js"))
		expectedSingle := "'use strict';function hello(){console.log('hola')}"
		expectedDouble := "\"use strict\";function hello(){console.log(\"hola\")}"
		if string(content) != expectedSingle && string(content) != expectedDouble {
			t.Errorf("JS minificado incorrecto:\nEsperado: %s o %s\nObtenido: %s", expectedSingle, expectedDouble, content)
		}
	})
}
