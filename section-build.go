package godev

import (
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
	wasmLogger := h.tui.AddLogger("WASM", false, colorPurpleMedium, sectionBuild)
	serverLogger := h.tui.AddLogger("SERVER", false, colorBlueMedium, sectionBuild)
	assetsLogger := h.tui.AddLogger("ASSETS", false, colorGreenMedium, sectionBuild)
	watchLogger := h.tui.AddLogger("WATCH", false, colorYellowMedium, sectionBuild)
	configLogger := h.tui.AddLogger("CONFIG", true, colorTealMedium, sectionBuild)
	browserLogger := h.tui.AddLogger("BROWSER", true, colorPinkMedium, sectionBuild)

	// CONFIG
	h.config = NewConfig(h.rootDir, configLogger) // Use the provided logger
	// ✅ No scanning needed - using conventional paths

	//SERVER
	h.serverHandler = goserver.New(&goserver.Config{
		AppRootDir:                  h.rootDir,
		RootFolder:                  filepath.Join(h.rootDir, h.config.GetWebFilesFolder(), "appserver"),
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
		WebFilesRootRelative: filepath.Join(h.config.GetWebFilesFolder(), "webclient"),
		WebFilesSubRelative:  h.config.GetPublicFolder(),
		Logger:               wasmLogger,
	})

	//ASSETS
	h.assetsHandler = NewAssetMin(&AssetConfig{
		ThemeFolder: func() string {
			return path.Join(h.rootDir, h.config.GetWebFilesFolder(), "webclient", "ui")
		},
		WebFilesFolder: func() string {
			return path.Join(h.rootDir, h.config.GetOutputStaticsDirectory())
		},
		Logger:                  assetsLogger,
		GetRuntimeInitializerJS: func() (string, error) { return "", nil },
	})

	// BROWSER
	h.browser = devbrowser.New(h.config, h.tui, h.exitChan, browserLogger)

	// WATCHER
	h.watcher = devwatch.New(&devwatch.WatchConfig{
		AppRootDir:         h.config.GetRootDir(),
		FilesEventHandlers: []devwatch.FilesEventHandlers{h.assetsHandler, h.wasmHandler, h.serverHandler},
		FolderEvents:       nil, // ✅ No dynamic folder event handling needed
		BrowserReload:      h.browser.Reload,
		Logger:             watchLogger,
		ExitChan:           h.exitChan,
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
		// override the watcher callback
		h.watcher.BrowserReload = h.pendingBrowserReload
		// clear pending to avoid accidental reuse
		h.pendingBrowserReload = nil
	}

	// Agregar manejadores que requieren interacción del desarrollador
	// BROWSER
	h.tui.AddHandler(h.browser, time.Millisecond*500, colorPinkMedium, sectionBuild)
	// WASM compilar wasm de forma dinámica
	h.tui.AddHandler(h.wasmHandler, time.Millisecond*500, colorPurpleMedium, sectionBuild)

}
