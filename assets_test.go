package godev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time" // Importar time

	"github.com/stretchr/testify/assert" // Importar testify
)

func TestUpdateFileOnDisk(t *testing.T) {
	// Configurar entorno de prueba
	rootDir := "test"
	webDir := filepath.Join(rootDir, "assetTest")
	defer os.RemoveAll(webDir)

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

	t.Run("check archivos theme CSS", func(t *testing.T) {
		assetsHandler.cssHandler.ClearMemoryFiles()
		os.Remove(styleCssPath)
		defer os.Remove(styleCssPath)

		sliceFiles := []struct {
			fileName string
			path     string
			content  string
		}{
			{"module.css", filepath.Join(webDir, "module.css"), ".test { color: red; }"},
			{"theme.css", filepath.Join(themeDir, "theme.css"), ":root { --primary: #ffffff; }"},
		}

		// create files
		for _, file := range sliceFiles {
			if err := os.WriteFile(file.path, []byte(file.content), 0644); err != nil {
				t.Fatal(err)
			}
		}

		// run event
		for _, file := range sliceFiles {
			if err := assetsHandler.NewFileEvent(file.fileName, ".css", file.path, "write"); err != nil {
				t.Fatal(err)
			}
		}

		// Verificar archivo generado
		if _, err := os.Stat(styleCssPath); os.IsNotExist(err) {
			t.Fatal("Archivo CSS no generado")
		}

		// Verificar contenido contenido theme debe estar primero
		content, _ := os.ReadFile(styleCssPath)
		if string(content) != ":root{--primary:#ffffff}.test{color:red}" {
			t.Fatalf("Contenido CSS minificado incorrecto: [%s]", content)
		}

		// remove files
		for _, file := range sliceFiles {
			os.Remove(file.path)
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

func TestAssetWatcherScenario(t *testing.T) {
	// --- Setup ---
	rootDir := "test"
	webDir := filepath.Join(rootDir, "assetWatcherTest")
	defer os.RemoveAll(webDir) // Limpieza al final

	publicDir := filepath.Join(webDir, "public")
	themeDir := filepath.Join(webDir, "theme")

	// Crear directorios
	for _, dir := range []string{webDir, publicDir, themeDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Error creando directorio de prueba: %v", err)
		}
	}

	// Configuración mock del AssetsHandler
	config := &AssetsConfig{
		ThemeFolder:    func() string { return themeDir },
		WebFilesFolder: func() string { return publicDir },
		Print: func(messages ...any) {
			// fmt.Println(messages...) // Descomentar para debug
		},
		JavascriptForInitializing: func() (string, error) {
			return "/*init*/", nil // Código JS inicial simple
		},
	}
	assetsHandler := NewAssetsCompiler(config)
	mainJsPath := filepath.Join(publicDir, assetsHandler.jsHandler.fileOutputName)

	// --- Test Steps ---

	// 1. Crear archivo "new file.txt" (no debería ser procesado)
	t.Run("Step 1: Create .txt file", func(t *testing.T) {
		txtFileName := "new file.txt"
		txtFilePath := filepath.Join(themeDir, txtFileName)
		txtContent := []byte("Este es un archivo de texto.")

		err := os.WriteFile(txtFilePath, txtContent, 0644)
		assert.NoError(t, err, "Error al escribir archivo .txt inicial")

		// Simular evento 'create' (aunque NewFileEvent lee el archivo, el tipo .txt debería ser ignorado)
		// Usamos 'write' para asegurar que writeOnDisk se active si fuera necesario, aunque no debería para .txt
		err = assetsHandler.NewFileEvent(txtFileName, ".txt", txtFilePath, "write")
		// Esperamos un error específico de extensión no soportada o ningún error si simplemente lo ignora
		// El código actual devuelve error, así que lo verificamos
		assert.Error(t, err, "Se esperaba un error para la extensión .txt")
		assert.Contains(t, err.Error(), "extension: .txt not found")

		// Verificar que main.js NO existe o está vacío
		_, err = os.Stat(mainJsPath)
		assert.True(t, os.IsNotExist(err), "main.js no debería existir después de crear un .txt")
	})

	// 2. Renombrar a .js y editar contenido (debería procesarse)
	t.Run("Step 2: Rename to .js and write content", func(t *testing.T) {
		txtFilePath := filepath.Join(themeDir, "new file.txt")
		jsFileName1 := "file1.js"
		jsFilePath1 := filepath.Join(themeDir, jsFileName1)
		jsContent1 := []byte("console.log('Archivo 1');")

		// Renombrar (simulado borrando y creando/escribiendo)
		err := os.Remove(txtFilePath)
		// Ignoramos el error si el archivo no existe (puede pasar si el paso 1 falló o limpió)
		if err != nil && !os.IsNotExist(err) {
			t.Logf("Advertencia: No se pudo borrar %s: %v", txtFilePath, err)
		}

		err = os.WriteFile(jsFilePath1, jsContent1, 0644)
		assert.NoError(t, err, "Error al escribir archivo .js inicial (file1.js)")

		// Simular evento 'write' para el nuevo archivo .js
		err = assetsHandler.NewFileEvent(jsFileName1, ".js", jsFilePath1, "write")
		assert.NoError(t, err, "Error al procesar el evento 'write' para file1.js")

		// Verificar que main.js existe y tiene el contenido esperado
		contentBytes, err := os.ReadFile(mainJsPath)
		assert.NoError(t, err, "Error al leer main.js después del paso 2")
		content := string(contentBytes)
		// Ajustado: Esperamos 'use strict'; y el código minificado, sin el comentario /*init*/ y sin ; final
		expectedContent := `"use strict";console.log("Archivo 1")`
		assert.Contains(t, content, expectedContent, "El contenido de main.js no es el esperado después del paso 2")
	})

	// 3. Crear otro archivo .js (debería añadirse al output)
	t.Run("Step 3: Create second .js file", func(t *testing.T) {
		jsFileName2 := "file2.js"
		jsFilePath2 := filepath.Join(themeDir, jsFileName2)
		jsContent2 := []byte("function saludar() { alert('Hola!'); }")

		err := os.WriteFile(jsFilePath2, jsContent2, 0644)
		assert.NoError(t, err, "Error al escribir el segundo archivo .js (file2.js)")

		// Simular evento 'write' para el segundo archivo .js
		err = assetsHandler.NewFileEvent(jsFileName2, ".js", jsFilePath2, "write")
		assert.NoError(t, err, "Error al procesar el evento 'write' para file2.js")

		// Verificar que main.js contiene ambos contenidos
		contentBytes, err := os.ReadFile(mainJsPath)
		assert.NoError(t, err, "Error al leer main.js después del paso 3")
		content := string(contentBytes)

		// El orden esperado es themeFiles primero, luego moduleFiles. Ambos están en themeDir.
		// El orden dentro de themeFiles/moduleFiles no está garantizado explícitamente en el código,
		// pero asumimos que se añaden al final.
		expectedContent1 := `console.log("Archivo 1")`           // Ajustado: sin ; final
		expectedContent2 := `function saludar(){alert("Hola!")}` // Contenido minificado esperado
		expectedStart := `"use strict";`                         // Ajustado: Esperamos 'use strict';

		assert.Contains(t, content, expectedStart, "Falta el código inicial ('use strict';) en main.js después del paso 3")
		assert.Contains(t, content, expectedContent1, "Falta el contenido de file1.js en main.js después del paso 3")
		assert.Contains(t, content, expectedContent2, "Falta el contenido de file2.js en main.js después del paso 3")
		// Verificar que no haya duplicados obvios (esto es una verificación simple)
		assert.Equal(t, 1, strings.Count(content, expectedContent1), "Contenido de file1.js duplicado")
		assert.Equal(t, 1, strings.Count(content, expectedContent2), "Contenido de file2.js duplicado")

	})

	// 4. Editar el contenido del archivo 1 (debería actualizarse sin duplicar)
	t.Run("Step 4: Edit first .js file", func(t *testing.T) {
		jsFileName1 := "file1.js"
		jsFilePath1 := filepath.Join(themeDir, jsFileName1)
		updatedJsContent1 := []byte("console.warn('Archivo 1 actualizado');") // Nuevo contenido

		// Esperar un poco para asegurar que el timestamp del archivo cambie
		time.Sleep(50 * time.Millisecond)

		err := os.WriteFile(jsFilePath1, updatedJsContent1, 0644)
		assert.NoError(t, err, "Error al actualizar el contenido de file1.js")

		// Simular evento 'write' para el archivo actualizado
		err = assetsHandler.NewFileEvent(jsFileName1, ".js", jsFilePath1, "write")
		assert.NoError(t, err, "Error al procesar el evento 'write' para file1.js actualizado")

		// Verificar el contenido final de main.js
		contentBytes, err := os.ReadFile(mainJsPath)
		assert.NoError(t, err, "Error al leer main.js después del paso 4")
		content := string(contentBytes)

		// originalContent1 ya no se usa, se comprueba directamente en NotContains
		updatedContent1 := `console.warn("Archivo 1 actualizado")` // Contenido minificado esperado (asumiendo que tampoco tendrá ;)
		content2 := `function saludar(){alert("Hola!")}`
		expectedStart := `"use strict";` // Ajustado: Esperamos 'use strict';

		assert.Contains(t, content, expectedStart, "Falta el código inicial ('use strict';) en main.js después del paso 4")
		assert.Contains(t, content, updatedContent1, "Falta el contenido actualizado de file1.js en main.js después del paso 4")
		assert.Contains(t, content, content2, "Falta el contenido de file2.js en main.js después del paso 4")
		// Ajustado: Verificar que el contenido original SIN punto y coma no esté presente
		assert.NotContains(t, content, `console.log("Archivo 1")`, "El contenido original de file1.js no debería estar presente después del paso 4")

		// Verificar que no haya duplicados
		// Ajustado: Contar el contenido actualizado SIN punto y coma
		assert.Equal(t, 1, strings.Count(content, `console.warn("Archivo 1 actualizado")`), "Contenido actualizado de file1.js duplicado")
		assert.Equal(t, 1, strings.Count(content, content2), "Contenido de file2.js duplicado después del paso 4")
	})
}
