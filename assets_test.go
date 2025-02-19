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
	rootDir := "test"
	webDir := filepath.Join(rootDir, "assetTestApp")
	// defer os.RemoveAll(webDir)

	publicDir := filepath.Join(webDir, "public")
	themeDir := filepath.Join(webDir, "theme")

	// Crear directorios si no existen
	for _, dir := range []string{webDir, publicDir, themeDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Error creando directorio de build: %v", err)
		}
	}

	// Configuración mock del compilador
	config := &AssetsConfig{
		ThemeFolder:    func() string { return themeDir },
		WebFilesFolder: func() string { return publicDir },
		Print: func(messages ...any) {
			fmt.Println(messages...)
		},
		JavascriptForInitializing: func() (string, error) {
			return "function init() { return 'test' };", nil
		},
	}
	assetsHandler := NewAssetsCompiler(config)
	styleCssPath := filepath.Join(publicDir, assetsHandler.cssHandler.fileOutputName)
	mainJsPath := filepath.Join(publicDir, assetsHandler.jsHandler.fileOutputName)

	t.Run("Verify writeOnDisk behavior", func(t *testing.T) {
		assetsHandler.cssHandler.ClearMemoryFiles()
		os.Remove(styleCssPath)
		fileName := "write_test.css"
		cssPath := filepath.Join(themeDir, fileName)
		defer os.Remove(cssPath)

		// Create initial file with create event
		os.WriteFile(cssPath, []byte(".create { color: blue; }"), 0644)
		if err := assetsHandler.NewFileEvent(fileName, ".css", cssPath, "create"); err != nil {
			t.Fatal(err)
		}

		// Verify file is not written on create event
		content, err := os.ReadFile(styleCssPath)
		if err == nil || string(content) != "" {
			t.Fatal(" expected file not written on create event")
		}

		// Update file with write event
		os.WriteFile(cssPath, []byte(".write { color: green; }"), 0644)
		if err := assetsHandler.NewFileEvent(fileName, ".css", cssPath, "write"); err != nil {
			t.Fatal(err)
		}

		// Verify file is written after write event
		content, err = os.ReadFile(styleCssPath)
		if err != nil {
			t.Fatal("File should exist after write event")
		}

		expected := ".write{color:green}"
		if string(content) != expected {
			t.Fatalf("\nexpected: %s\ngot: %s", expected, string(content))
		}

	})

	t.Run("DISABLED_Crear nuevo archivo CSS", func(t *testing.T) {
		assetsHandler.cssHandler.ClearMemoryFiles()
		fileName := "test.css"
		cssPath := filepath.Join(themeDir, fileName)
		defer os.Remove(styleCssPath)
		defer os.Remove(cssPath)
		event := "write"

		// Crear archivo CSS de prueba
		if err := os.WriteFile(cssPath, []byte(".test { color: red; }"), 0644); err != nil {
			t.Fatal(err)
		}

		// Ejecutar función bajo prueba
		if err := assetsHandler.NewFileEvent(fileName, ".css", cssPath, event); err != nil {
			t.Fatalf("Error inesperado: %v", err)
		}

		// Verificar archivo generado
		if _, err := os.Stat(styleCssPath); os.IsNotExist(err) {
			t.Fatal("Archivo CSS no generado")
		}

		// Verificar contenido minificado
		content, _ := os.ReadFile(styleCssPath)
		if string(content) != ".test{color:red}" {
			t.Fatalf("Contenido CSS minificado incorrecto: [%s]", content)
		}
	})

	t.Run("Actualizar archivo CSS existente", func(t *testing.T) {
		assetsHandler.cssHandler.ClearMemoryFiles()
		fileName := "existing.css"
		cssPath := filepath.Join(themeDir, fileName)
		defer os.Remove(styleCssPath)
		defer os.Remove(cssPath)

		// Crear archivo inicial
		os.WriteFile(cssPath, []byte(".old { padding: 1px; }"), 0644)
		assetsHandler.NewFileEvent(fileName, ".css", cssPath, "create")

		// Actualizar contenido
		os.WriteFile(cssPath, []byte(".new { margin: 2px; }"), 0644)
		if err := assetsHandler.NewFileEvent(fileName, ".css", cssPath, "write"); err != nil {
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
		defer os.Remove(styleCssPath)
		fileName := "no_existe.css"
		err := assetsHandler.NewFileEvent(fileName, ".css", "", "write")
		if err == nil {
			t.Fatal("Se esperaba error por archivo no encontrado")
		}
	})

	t.Run("Extensión inválida", func(t *testing.T) {
		defer os.Remove(styleCssPath)
		fileName := "archivo.txt"
		filePath := filepath.Join(themeDir, fileName)
		err := assetsHandler.NewFileEvent(fileName, ".txt", filePath, "write")
		if err == nil {
			t.Fatal("Se esperaba error por extensión inválida")
		}
	})

	t.Run("Crear archivo JS básico", func(t *testing.T) {
		assetsHandler.jsHandler.ClearMemoryFiles()

		fileName1 := "test.js"
		fileName2 := "test2.js"
		jsPath := filepath.Join(themeDir, fileName1)
		jsPath2 := filepath.Join(themeDir, fileName2)
		defer os.Remove(jsPath)
		defer os.Remove(jsPath2)
		defer os.Remove(mainJsPath)

		os.WriteFile(jsPath, []byte(`// Test\nfunction hello() { console.log("hola") }
		let x = 10;`), 0644)
		os.WriteFile(jsPath2, []byte(`// Test2\nfunction bye() { console.log("adios") }
		let y = 20;`), 0644)

		if err := assetsHandler.NewFileEvent(fileName1, ".js", jsPath, "create"); err != nil {
			t.Fatal(err)
		}
		if err := assetsHandler.NewFileEvent(fileName2, ".js", jsPath2, "write"); err != nil {
			t.Fatal(err)
		}

		got, _ := os.ReadFile(filepath.Join(publicDir, assetsHandler.jsHandler.fileOutputName))
		expected := `"use strict";function init(){return"test"}let x=10,y=20`
		if string(got) != expected {
			t.Fatalf("\nJS minificado incorrecto:\nexpected: \n[%s]\n\ngot: \n[%s]", expected, got)
		}

	})

}
