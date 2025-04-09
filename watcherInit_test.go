package godev

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Import types from other files in the package
// These types are defined in:
// - assets.go: AssetsConfig, AssetsHandler
// - watcher.go: WatchConfig, WatchHandler

// setupWatcherAssetsTest encapsulates the common setup logic for watcher assets tests.
// It returns the necessary variables for the test execution and teardown.
func setupWatcherAssetsTest(t *testing.T) (
	tmpDir string,
	themeDir string,
	publicDir string,
	assetsHandler *AssetsHandler,
	watcher *WatchHandler,
	exitChan chan bool,
	logBuf *bytes.Buffer,
	logBufMu *sync.Mutex, // Mutex for logBuf
	wg *sync.WaitGroup,
	outputJsPath string,
) {
	t.Helper() // Mark this as a helper function

	// --- Setup ---
	tmpDir = t.TempDir()
	t.Logf("Directorio temporal de prueba: %s", tmpDir) // Log temp dir

	webDir := filepath.Join(tmpDir, "web")
	publicDir = filepath.Join(webDir, "public")
	themeDir = filepath.Join(webDir, "theme")

	require.NoError(t, os.MkdirAll(publicDir, 0755))
	require.NoError(t, os.MkdirAll(themeDir, 0755)) // Crear themeDir

	exitChan = make(chan bool)
	logBuf = new(bytes.Buffer) // Use new for pointer
	logBufMu = new(sync.Mutex) // Initialize the mutex

	// Define the print function that uses the mutex
	printFunc := func(messages ...any) {
		logBufMu.Lock()
		defer logBufMu.Unlock()
		logBuf.WriteString(fmt.Sprintln(messages...))
	}

	assetsCfg := &AssetsConfig{
		ThemeFolder:               func() string { return themeDir },
		WebFilesFolder:            func() string { return publicDir },
		Print:                     printFunc, // Use the mutex-protected print function
		JavascriptForInitializing: func() (string, error) { return "", nil },
	}

	assetsHandler = NewAssetsCompiler(assetsCfg)
	assetsHandler.writeOnDisk = true

	watchCfg := &WatchConfig{
		AppRootDir:      tmpDir,
		FileEventAssets: assetsHandler,
		FileEventGO:     nil,
		FileEventWASM:   nil,
		FileTypeGO:      nil,
		BrowserReload:   func() error { return nil },
		Print:           printFunc, // Use the mutex-protected print function
		ExitChan:        exitChan,
		UnobservedFiles: assetsHandler.UnobservedFiles,
	}

	watcher = NewWatchHandler(watchCfg)

	wg = new(sync.WaitGroup) // Use new for pointer
	wg.Add(1)
	go func() {
		watcher.FileWatcherStart(wg) // Pasar wg aquí
	}()

	// Espera corta para que el watcher inicie
	time.Sleep(50 * time.Millisecond)

	outputJsPath = filepath.Join(publicDir, "main.js") // Ruta del archivo de salida

	return // Return named variables
}
