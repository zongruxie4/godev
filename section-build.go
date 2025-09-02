package godev

import (
	"fmt"
	"path"
	"path/filepath"
	"time"

	. "github.com/cdvelop/assetmin"
	"github.com/cdvelop/devbrowser"
	"github.com/cdvelop/devwatch"
	"github.com/cdvelop/goserver"
	"github.com/cdvelop/tinywasm"
)

func (h *handler) AddSectionBUILD() {

	// LDFlags      func() []string // eg: []string{"-X 'main.version=v1.0.0'","-X 'main.buildDate=2023-01-01'"}

	sectionBuild := h.tui.NewTabSection("BUILD", "Building and Compiling")

	// WRITERS
	wasmLogger := sectionBuild.NewLogger("WASM", false)
	serverLogger := sectionBuild.NewLogger("SERVER", false)
	assetsLogger := sectionBuild.NewLogger("ASSETS", false)
	watchLogger := sectionBuild.NewLogger("WATCH", false)
	configLogger := sectionBuild.NewLogger("CONFIG", true)
	browserLogger := sectionBuild.NewLogger("BROWSER", true)

	// CONFIG
	h.config = NewAutoConfig(h.rootDir, configLogger) // Use the provided logger
	// Scan initial architecture - this must happen before AddSectionBUILD
	h.config.ScanDirectoryStructure()

	//SERVER
	h.serverHandler = goserver.New(&goserver.Config{
		AppRootDir:                  h.rootDir,
		RootFolder:                  filepath.Join(h.rootDir, h.config.GetWebFilesFolder()),
		MainFileWithoutExtension:    "main.server",
		ArgumentsForCompilingServer: nil,
		ArgumentsToRunServer:        nil,
		PublicFolder:                h.config.GetPublicFolder(),
		AppPort:                     h.config.GetServerPort(),
		Logger:                      serverLogger,
		ExitChan:                    h.exitChan,
	})

	//WASM
	h.wasmHandler = tinywasm.New(&tinywasm.Config{
		AppRootDir:           h.rootDir,
		WebFilesRootRelative: h.config.GetWebFilesFolder(),
		WebFilesSubRelative:  h.config.GetPublicFolder(),
		Logger:               wasmLogger,
	})

	//ASSETS
	h.assetsHandler = NewAssetMin(&AssetConfig{
		ThemeFolder: func() string {
			return path.Join(h.rootDir, h.config.GetWebFilesFolder(), "theme")
		},
		WebFilesFolder: func() string {
			return path.Join(h.rootDir, h.config.GetOutputStaticsDirectory())
		},
		Logger:                  assetsLogger,
		GetRuntimeInitializerJS: h.wasmHandler.JavascriptForInitializing,
	})

	// BROWSER
	h.browser = devbrowser.New(h.config, h.tui, h.exitChan, browserLogger)

	// WATCHER
	h.watcher = devwatch.New(&devwatch.WatchConfig{
		AppRootDir:      h.config.GetRootDir(),
		FileEventAssets: h.assetsHandler,
		FilesEventGO:    []devwatch.GoFileHandler{h.serverHandler, h.wasmHandler},
		FolderEvents:    h.config, // Architecture detection for directory changes
		BrowserReload:   h.browser.Reload,
		Logger:          watchLogger,
		ExitChan:        h.exitChan,
		UnobservedFiles: func() []string {

			uf := []string{
				".git",
				".gitignore",
				".vscode",
				".exe",
				".log",
				"_test.go",
			}

			uf = append(uf, h.assetsHandler.UnobservedFiles()...)
			uf = append(uf, h.wasmHandler.UnobservedFiles()...)
			uf = append(uf, h.serverHandler.UnobservedFiles()...)
			return uf
		},
	})

	// If tests set a pending browser reload callback before the watcher was
	// created, apply it now so tests can observe reload calls.
	if h.pendingBrowserReload != nil {
		fmt.Println("Applying pendingBrowserReload to watcher")
		// override the watcher callback
		h.watcher.BrowserReload = h.pendingBrowserReload
		// clear pending to avoid accidental reuse
		h.pendingBrowserReload = nil
	}

	// Agregar manejadores que requieren interacción del desarrollador
	// BROWSER
	sectionBuild.AddExecutionHandler(h.browser, time.Millisecond*500)
	// WASM compilar wasm de forma dinámica
	sectionBuild.AddEditHandler(h.wasmHandler, time.Millisecond*500)

}
