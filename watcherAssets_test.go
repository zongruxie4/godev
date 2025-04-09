package godev

import (
	"os"
	"path/filepath"
	"strings"
	_ "sync" // Use blank identifier to force import
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatcherAssetsIntegration(t *testing.T) {
	// --- Setup using helper ---
	// Call the setup function from watcherInit_test.go
	// We only need a subset of the returned variables directly in this test function's scope.
	// _ indicates variables we don't need to reference directly here (like tmpDir, assetsHandler, watcher).
	_, themeDir, _, _, _, exitChan, logBuf, logBufMu, wg, outputJsPath := setupWatcherAssetsTest(t) // Receive logBufMu

	// Helper function to safely get log buffer content
	getLogContent := func() string {
		logBufMu.Lock()
		defer logBufMu.Unlock()
		return logBuf.String()
	}

	// Helper function to safely reset log buffer
	resetLogBuffer := func() {
		logBufMu.Lock()
		defer logBufMu.Unlock()
		logBuf.Reset()
	}

	// --- Test Steps ---

	// 1. Crear archivo "new file.txt" (no debería ser procesado)
	t.Run("Step 1: Create .txt file", func(t *testing.T) {
		txtFileName := "new file.txt"
		txtFilePath := filepath.Join(themeDir, txtFileName)
		txtContent := []byte("Este es un archivo de texto.")

		t.Logf("Step 1: Creando %s", txtFilePath)
		require.NoError(t, os.WriteFile(txtFilePath, txtContent, 0644), "Error al escribir archivo .txt inicial")

		// Esperar posible procesamiento (aunque no debería ocurrir)
		time.Sleep(150 * time.Millisecond) // Aumentar espera por si acaso

		// Verificar que main.js NO existe
		_, err := os.Stat(outputJsPath)
		require.True(t, os.IsNotExist(err), "main.js no debería existir después de crear un .txt. Logs:\n%s", getLogContent())
		resetLogBuffer() // Limpiar buffer para el siguiente paso
	})

	// 2. Renombrar a .js y escribir contenido (debería procesarse)
	t.Run("Step 2: Rename to .js and write content", func(t *testing.T) {
		txtFilePath := filepath.Join(themeDir, "new file.txt")
		jsFileName1 := "file1.js"
		jsFilePath1 := filepath.Join(themeDir, jsFileName1)
		jsContent1 := []byte("console.log('Archivo 1');")

		t.Logf("Step 2: Eliminando %s", txtFilePath)
		require.NoError(t, os.Remove(txtFilePath), "Error al eliminar .txt")
		// No es necesaria una espera aquí, el watcher debería detectar REMOVE

		// Crear directamente el archivo .js con su contenido
		t.Logf("Step 2: Creando y escribiendo contenido en %s", jsFilePath1)
		require.NoError(t, os.WriteFile(jsFilePath1, jsContent1, 0644), "Error al escribir file1.js")

		// Esperar procesamiento del evento WRITE
		time.Sleep(200 * time.Millisecond) // Espera más larga para asegurar procesamiento

		// Verificar que main.js existe y tiene el contenido esperado
		require.FileExists(t, outputJsPath, "main.js debería existir después del paso 2. Logs:\n%s", getLogContent())
		contentBytes, err := os.ReadFile(outputJsPath)
		require.NoError(t, err, "Error al leer main.js después del paso 2")
		content := string(contentBytes)
		expectedContent := `"use strict";console.log("Archivo 1")` // Sin ; final
		require.Contains(t, content, expectedContent, "El contenido de main.js no es el esperado después del paso 2. Logs:\n%s", getLogContent())
		resetLogBuffer()
	})

	// 3. Crear otro archivo .js (debería añadirse al output)
	t.Run("Step 3: Create second .js file", func(t *testing.T) {
		jsFileName2 := "file2.js"
		jsFilePath2 := filepath.Join(themeDir, jsFileName2)
		jsContent2 := []byte("function saludar() { alert('Hola!'); }")

		t.Logf("Step 3: Creando %s", jsFilePath2)
		require.NoError(t, os.WriteFile(jsFilePath2, jsContent2, 0644), "Error al escribir file2.js")

		// Esperar procesamiento
		time.Sleep(200 * time.Millisecond)

		// Verificar que main.js contiene ambos contenidos sin duplicar
		require.FileExists(t, outputJsPath, "main.js debería existir después del paso 3. Logs:\n%s", getLogContent())
		contentBytes, err := os.ReadFile(outputJsPath)
		require.NoError(t, err, "Error al leer main.js después del paso 3")
		content := string(contentBytes)

		expectedContent1 := `console.log("Archivo 1")`
		expectedContent2 := `function saludar(){alert("Hola!")}`
		expectedStart := `"use strict";`

		require.Contains(t, content, expectedStart, "Falta 'use strict'; en main.js después del paso 3. Logs:\n%s", getLogContent())
		require.Contains(t, content, expectedContent1, "Falta contenido de file1.js en main.js después del paso 3. Logs:\n%s", getLogContent())
		require.Contains(t, content, expectedContent2, "Falta contenido de file2.js en main.js después del paso 3. Logs:\n%s", getLogContent())

		// Verificar no duplicados (simple)
		require.Equal(t, 1, strings.Count(content, expectedContent1), "Contenido de file1.js duplicado después del paso 3. Logs:\n%s", getLogContent())
		require.Equal(t, 1, strings.Count(content, expectedContent2), "Contenido de file2.js duplicado después del paso 3. Logs:\n%s", getLogContent())
		resetLogBuffer()
	})

	// 4. Editar el contenido del archivo 1 (debería actualizarse sin duplicar)
	t.Run("Step 4: Edit first .js file", func(t *testing.T) {
		jsFilePath1 := filepath.Join(themeDir, "file1.js")
		updatedJsContent1 := []byte("console.warn('Archivo 1 actualizado');")

		t.Logf("Step 4: Actualizando %s", jsFilePath1)
		require.NoError(t, os.WriteFile(jsFilePath1, updatedJsContent1, 0644), "Error al actualizar file1.js")

		// Esperar procesamiento
		time.Sleep(200 * time.Millisecond)

		// Verificar contenido final
		require.FileExists(t, outputJsPath, "main.js debería existir después del paso 4. Logs:\n%s", getLogContent())
		contentBytes, err := os.ReadFile(outputJsPath)
		require.NoError(t, err, "Error al leer main.js después del paso 4")
		content := string(contentBytes)

		originalContent1 := `console.log("Archivo 1")`
		updatedContent1 := `console.warn("Archivo 1 actualizado")`
		content2 := `function saludar(){alert("Hola!")}`
		expectedStart := `"use strict";`

		require.Contains(t, content, expectedStart, "Falta 'use strict'; en main.js después del paso 4. Logs:\n%s", getLogContent())
		require.Contains(t, content, updatedContent1, "Falta contenido actualizado de file1.js en main.js después del paso 4. Logs:\n%s", getLogContent())
		require.Contains(t, content, content2, "Falta contenido de file2.js en main.js después del paso 4. Logs:\n%s", getLogContent())
		require.NotContains(t, content, originalContent1, "Contenido original de file1.js no debería estar presente después del paso 4. Logs:\n%s", getLogContent())

		// Verificar no duplicados
		require.Equal(t, 1, strings.Count(content, updatedContent1), "Contenido actualizado de file1.js duplicado después del paso 4. Logs:\n%s", getLogContent())
		require.Equal(t, 1, strings.Count(content, content2), "Contenido de file2.js duplicado después del paso 4. Logs:\n%s", getLogContent())
		resetLogBuffer()
	})

	// --- Teardown ---
	t.Log("Deteniendo watcher...")
	close(exitChan)
	wg.Wait() // Esperar a que la goroutine del watcher termine limpiamente

	t.Log("Test de integración completado.")
	// Imprimir logs finales solo si el test falla (require maneja esto implícitamente)
	// Si quieres ver los logs siempre, descomenta la siguiente línea:
	// t.Log("Logs finales del watcher:\n", getLogContent())
}
