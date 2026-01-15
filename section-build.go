package app

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/client"
	"github.com/tinywasm/devbrowser"
	"github.com/tinywasm/devwatch"
	"github.com/tinywasm/server"
)

func (h *handler) AddSectionBUILD() any {

	// LDFlags      func() []string // eg: []string{"-X 'main.version=v1.0.0'","-X 'main.buildDate=2023-01-01'"}

	sectionBuild := h.tui.NewTabSection("BUILD", "Building and Compiling")

	// WRITERS
	// CONFIG
	if h.config == nil {
		h.config = NewConfig(h.rootDir, nil) // Logger will be injected via AddHandler
	}

	// 1. WASM Client - Core logic handlers
	h.wasmClient = client.New(&client.Config{
		SourceDir: h.config.CmdWebClientDir,
		OutputDir: h.config.WebPublicDir,
		Database:  h.db,
	})

	//ASSETS
	h.assetsHandler = assetmin.NewAssetMin(&assetmin.Config{
		OutputDir: filepath.Join(h.rootDir, h.config.WebPublicDir()),
		GetRuntimeInitializerJS: func() (string, error) {
			return h.wasmClient.JavascriptForInitializing()
		},
		AppName: h.frameworkName,
	})

	//SERVER
	h.serverHandler = server.New(&server.Config{
		AppRootDir:                  h.rootDir,
		SourceDir:                   h.config.CmdAppServerDir(),
		OutputDir:                   h.config.DeployAppServerDir(),
		MainInputFile:               h.config.ServerFileName(),
		Routes:                      []func(*http.ServeMux){h.assetsHandler.RegisterRoutes, h.wasmClient.RegisterRoutes},
		ArgumentsForCompilingServer: func() []string { return []string{} },
		ArgumentsToRunServer: func() []string {
			return []string{
				"-public-dir=" + filepath.Join(h.rootDir, h.config.WebPublicDir()),
				"-port=" + h.config.ServerPort(),
			}
		},
		AppPort:              h.config.ServerPort(),
		DisableGlobalCleanup: TestMode,
		ExitChan:             h.exitChan,
	})

	// BROWSER
	h.browser = devbrowser.New(h.config, h.tui, h.db, h.exitChan)

	// Create browser reload wrapper that supports test hooks
	browserReloadFunc := func() error {
		// Check if test hook is set (for testing)
		if f := GetInitialBrowserReloadFunc(); f != nil {
			return f()
		}
		// Use real browser reload (production)
		return h.browser.Reload()
	}

	// WATCHER
	h.watcher = devwatch.New(&devwatch.WatchConfig{
		AppRootDir:         h.config.RootDir(),
		FilesEventHandlers: []devwatch.FilesEventHandlers{h.wasmClient, h.serverHandler, h.assetsHandler},
		FolderEvents:       nil, // ✅ No dynamic folder event handling needed
		BrowserReload:      browserReloadFunc,
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
			uf = append(uf, h.wasmClient.UnobservedFiles()...)
			uf = append(uf, h.serverHandler.UnobservedFiles()...)
			return uf
		},
	})
	h.watcher.SetShouldWatch(h.isPartOfProject)

	// HANDLER REGISTRATION (Loggers)
	h.tui.AddHandler(h.wasmClient, 0, colorPurpleMedium, sectionBuild)
	h.tui.AddHandler(h.serverHandler, 0, colorBlueMedium, sectionBuild)
	h.tui.AddHandler(h.assetsHandler, 0, colorGreenMedium, sectionBuild)
	h.tui.AddHandler(h.watcher, 0, colorYellowMedium, sectionBuild)
	h.tui.AddHandler(h.config, 0, colorTealMedium, sectionBuild)
	h.tui.AddHandler(h.browser, 0, colorPinkMedium, sectionBuild)

	// Wire up TinyWasm to AssetMin
	h.wasmClient.OnWasmExecChange = func() {
		// Notify AssetMin to refresh JS assets (this will pull the new initializer JS)
		h.assetsHandler.RefreshAsset(".js")
		//h.wasmClient.Logger(" DEBUG: Refreshed script.js via AssetMin")

		// Reload the browser to apply changes
		if err := h.browser.Reload(); err != nil {
			h.wasmClient.Logger("Error reloading browser:", err)
		}
	}

	// Agregar manejadores que requieren interacción del desarrollador
	// WORK MODES (Build and Server)
	h.tui.AddHandler(&BuildModeOnDisk{h: h}, time.Millisecond*500, colorTealMedium, sectionBuild)
	h.tui.AddHandler(h.NewServerModeHandler(), time.Millisecond*500, colorBlueMedium, sectionBuild)

	return sectionBuild
}
