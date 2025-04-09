package godev

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync" // Import sync
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatcherAssetsIntegration(t *testing.T) {
	// Configurar entorno temporal
	tmpDir := t.TempDir()

	// Crear estructura de directorios
	webDir := filepath.Join(tmpDir, "web")
	publicDir := filepath.Join(webDir, "public")
	themeJsDir := filepath.Join(webDir, "theme", "js")

	require.NoError(t, os.MkdirAll(publicDir, 0755))
	require.NoError(t, os.MkdirAll(themeJsDir, 0755))

	// Configurar manejadores
	exitChan := make(chan bool)
	var logBuf bytes.Buffer

	// Configuración mínima para AssetsHandler
	assetsCfg := &AssetsConfig{
		ThemeFolder: func() string { return filepath.Join(webDir, "theme") },
		// Make WebFilesFolder return the actual public output directory for the test
		WebFilesFolder: func() string { return publicDir },
		Print:          func(messages ...any) { logBuf.WriteString(fmt.Sprint(messages...)); logBuf.WriteByte('\n') },
		// Simplify init code for debugging duplication issue
		JavascriptForInitializing: func() (string, error) { return "", nil },
	}

	assetsHandler := NewAssetsCompiler(assetsCfg)
	assetsHandler.writeOnDisk = true // Force disk writing for the test

	// Configuración para WatchHandler
	watchCfg := &WatchConfig{
		AppRootDir:      tmpDir,
		FileEventAssets: assetsHandler,
		FileEventGO:     nil,
		FileEventWASM:   nil,
		FileTypeGO:      nil,
		BrowserReload:   func() error { return nil },
		Writer:          &logBuf,
		ExitChan:        exitChan,
		UnobservedFiles: assetsHandler.UnobservedFiles, // Explicitly assign the function
	}

	watcher := NewWatchHandler(watchCfg)

	// Ejecutar watcher en goroutine
	var wg sync.WaitGroup // Add WaitGroup
	wg.Add(1)
	go func() {
		// wg.Done() is called inside FileWatcherStart
		watcher.FileWatcherStart(&wg) // Pass the WaitGroup pointer
	}()

	// Esperar a que el watcher esté listo
	time.Sleep(10 * time.Millisecond)

	// 1. Crear archivo main.js directamente (vacío inicialmente)
	mainJsPath := filepath.Join(themeJsDir, "main.js")
	require.NoError(t, os.WriteFile(mainJsPath, []byte(""), 0644)) // Create empty file first
	// Revert sleep to a reasonable value now that debouncer is disabled for diagnosis
	time.Sleep(10 * time.Millisecond) // Reverted from 600ms

	// 2. Escribir contenido en main.js
	jsContent := "console.log('Hello World');"
	require.NoError(t, os.WriteFile(mainJsPath, []byte(jsContent), 0644)) // Now write the actual content

	// Esperar procesamiento (mantener la espera larga para asegurar que el evento WRITE se procese)
	time.Sleep(150 * time.Millisecond) // Increased sleep duration falla si es menos de 150ms

	// Detener watcher
	close(exitChan)
	wg.Wait() // Wait for the watcher goroutine to finish

	// Verificar resultados
	outputJsPath := filepath.Join(publicDir, "main.js")
	// Check if file exists first, include logs on failure
	require.FileExists(t, outputJsPath, "Output file '%s' was not created. Logs:\n%s", outputJsPath, logBuf.String())

	// If file exists, proceed to read and check content
	outputContent, err := os.ReadFile(outputJsPath)
	require.NoError(t, err, "Failed to read existing output file '%s'. Logs:\n%s", outputJsPath, logBuf.String())

	// Use require.Equal for exact match comparison after minification
	expectedMinifiedContent := "\"use strict\";console.log(\"Hello World\")" // No semicolon at the end - minifier removes it
	require.Equal(t, expectedMinifiedContent, string(outputContent), "Output content mismatch. Logs:\n%s", logBuf.String())

	t.Log("Test completado exitosamente")
	t.Log("Logs del watcher:\n", logBuf.String())
}
