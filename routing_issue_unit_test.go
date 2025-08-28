package godev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cdvelop/devwatch"
	"github.com/cdvelop/godepfind"
	"github.com/stretchr/testify/require"
)

// TestGoHandlerRoutingIssue is a unit test that reproduces the specific issue
// where WASM handler without main.wasm.go incorrectly claims database/db.go
// when it should belong to the server handler.
//
// The issue occurs because godepfind.ThisFileIsMine doesn't validate if the
// mainInputFile (main.wasm.go) actually exists before claiming ownership of
// other files that could potentially depend on it.
//
// This test directly uses the DevWatch component to isolate the routing issue
// without the full godev application startup overhead.
func TestGoHandlerRoutingIssue(t *testing.T) {
	// 1. Crear un directorio temporal que represente el proyecto
	tmp := t.TempDir()

	// 2. Crear la estructura de carpetas: pwa (servidor) y database (paquete)
	serverDir := filepath.Join(tmp, "pwa")
	databaseDir := filepath.Join(tmp, "database")

	require.NoError(t, os.MkdirAll(serverDir, 0755))
	require.NoError(t, os.MkdirAll(databaseDir, 0755))

	// 3. Escribir el archivo go.mod básico para el módulo de prueba
	goModContent := `module testproject

go 1.21
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	// 4. Crear main.server.go que importa el paquete database (representa el server)
	serverMainPath := filepath.Join(serverDir, "main.server.go")
	serverContent := `package main

import "testproject/database"

func main() {

	database.Connect()

	printf("Server starting on port 4430")
}
`
	require.NoError(t, os.WriteFile(serverMainPath, []byte(serverContent), 0644))

	// 5. Crear database/db.go con una función Connect (archivo que debe pertenecer al servidor)
	dbPath := filepath.Join(databaseDir, "db.go")
	dbContent := `package database

func Connect() {
	println("Connected to database...")
}
`
	require.NoError(t, os.WriteFile(dbPath, []byte(dbContent), 0644))

	// 6. Preparar contadores para capturar llamadas registradas por cada handler
	var serverCalls []string
	var wasmCalls []string

	// 7. Crear un handler de servidor simulado (como goserver.New)
	serverHandler := &TestServerHandler{
		mainPath: "pwa/main.server.go",
		calls:    &serverCalls,
	}

	// 8. Crear un handler WASM simulado SIN main.wasm.go (simula el fallo)
	// Simulamos el comportamiento real de TinyWasm que construye dinámicamente la ruta
	wasmHandler := &TestWasmHandler{
		webFilesRootRelative: "pwa",          // Como en TinyWasm real
		mainInputFile:        "main.wasm.go", // Como en TinyWasm real
		calls:                &wasmCalls,
	}

	// 9. Construir DevWatch con ambos handlers (no se usa activamente aquí pero es contexto)
	logOutput := &strings.Builder{}
	watcher := devwatch.New(&devwatch.WatchConfig{
		AppRootDir:      tmp,
		FileEventAssets: &NoOpAssetHandler{}, // No relevante para este test
		FilesEventGO:    []devwatch.GoFileHandler{serverHandler, wasmHandler},
		FolderEvents:    &NoOpFolderHandler{}, // No relevante para este test
		BrowserReload:   func() error { return nil },
		Logger:          logOutput,
		ExitChan:        make(chan bool, 1),
	})

	// 10. Simular el evento sobre database/db.go (punto donde ocurre el routing)
	// En vez de iniciar el watcher se usa directamente el depFinder

	// 11. Obtener el buscador de dependencias que DevWatch usa internamente
	depFinder := godepfind.New(tmp)

	// 12. Comprobar la lógica de enrutamiento: ¿qué handler reclama database/db.go?
	t.Logf("Testing file ownership detection for database/db.go")

	// 13. Preguntarle al depFinder si el handler del servidor reclama db.go
	serverShouldClaim, err := depFinder.ThisFileIsMine(serverHandler.MainInputFileRelativePath(), dbPath, "write")
	require.NoError(t, err)
	t.Logf("Server handler (main: %s) claims db.go: %v", serverHandler.MainInputFileRelativePath(), serverShouldClaim)

	// 14. Preguntarle al depFinder si el handler WASM reclama db.go
	//     (se espera error/false porque main.wasm.go no existe)
	wasmShouldClaim, err := depFinder.ThisFileIsMine(wasmHandler.MainInputFileRelativePath(), dbPath, "write")
	if err != nil {
		t.Logf("WASM handler (main: %s) error claiming db.go: %v", wasmHandler.MainInputFileRelativePath(), err)
		wasmShouldClaim = false
	} else {
		t.Logf("WASM handler (main: %s) claims db.go: %v", wasmHandler.MainInputFileRelativePath(), wasmShouldClaim)
	}

	// 15. Analizar resultados y fallos esperados
	if wasmShouldClaim && !serverShouldClaim {
		t.Errorf("ISSUE REPRODUCED: WASM handler incorrectly claims database/db.go")
		t.Errorf("WASM handler main file: %s (does not exist)", wasmHandler.MainInputFileRelativePath())
		t.Errorf("Server handler main file: %s (imports testproject/database)", serverHandler.MainInputFileRelativePath())
		t.Errorf("Expected: Only server handler should claim db.go")
		t.Errorf("Actual: WASM handler claims db.go despite missing main.wasm.go")
	} else if wasmShouldClaim && serverShouldClaim {
		t.Errorf("ISSUE DETECTED: Both handlers claim database/db.go")
		t.Errorf("Expected: Only server handler should claim db.go")
		t.Errorf("Actual: Both handlers claim it")
	} else if serverShouldClaim && !wasmShouldClaim {
		t.Logf("SUCCESS: Only server handler correctly claims db.go")
	} else {
		t.Errorf("UNEXPECTED: Neither handler claims db.go")
		t.Errorf("Expected: Server handler should claim db.go (main.server.go imports database)")
		t.Errorf("Actual: Neither handler claims it")
	}

	// 16. Limpiar: señalizar salida del watcher simulado
	watcher.ExitChan <- true
}

// TestServerHandler simulates goserver.ServerHandler for testing
type TestServerHandler struct {
	mainPath string
	calls    *[]string
}

func (h *TestServerHandler) MainInputFileRelativePath() string {
	return h.mainPath
}

func (h *TestServerHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	call := "SERVER: " + event + " " + fileName + " " + filePath
	*h.calls = append(*h.calls, call)
	return nil
}

// TestWasmHandler simulates tinywasm.TinyWasm for testing
type TestWasmHandler struct {
	webFilesRootRelative string // Simula Config.WebFilesRootRelative de TinyWasm
	mainInputFile        string // Simula mainInputFile de TinyWasm ("main.wasm.go")
	calls                *[]string
}

func (h *TestWasmHandler) MainInputFileRelativePath() string {
	// Simula el comportamiento real de TinyWasm.MainInputFileRelativePath()
	// return path.Join(rootFolder, w.mainInputFile)
	return h.webFilesRootRelative + "/" + h.mainInputFile
}

func (h *TestWasmHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	call := "WASM: " + event + " " + fileName + " " + filePath
	*h.calls = append(*h.calls, call)
	return nil
}

// NoOpAssetHandler for testing (not relevant to this test)
type NoOpAssetHandler struct{}

func (h *NoOpAssetHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	return nil
}

// NoOpFolderHandler for testing (not relevant to this test)
type NoOpFolderHandler struct{}

func (h *NoOpFolderHandler) NewFolderEvent(folderName, path, event string) error {
	return nil
}
