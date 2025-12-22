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

func (h *handler) AddSectionBUILD() {

	// LDFlags      func() []string // eg: []string{"-X 'main.version=v1.0.0'","-X 'main.buildDate=2023-01-01'"}

	sectionBuild := h.tui.NewTabSection("BUILD", "Building and Compiling")

	// WRITERS
	wasmLogger := h.tui.AddLogger("WASM", true, colorPurpleMedium, sectionBuild)
	serverLogger := h.tui.AddLogger("SERVER", false, colorBlueMedium, sectionBuild)
	assetsLogger := h.tui.AddLogger("ASSETS", false, colorGreenMedium, sectionBuild)
	watchLogger := h.tui.AddLogger("WATCH", false, colorYellowMedium, sectionBuild)
	configLogger := h.tui.AddLogger("CONFIG", true, colorTealMedium, sectionBuild)
	browserLogger := h.tui.AddLogger("BROWSER", true, colorPinkMedium, sectionBuild)

	// CONFIG
	h.config = NewConfig(h.rootDir, configLogger) // Use the provided logger
	// ✅ No scanning needed - using conventional paths

	//WASM
	h.wasmClient = client.New(&client.Config{
		SourceDir: h.config.CmdWebClientDir(),
		OutputDir: h.config.WebPublicDir(),
		Logger:    wasmLogger,
		Store:     h.db,
		OnWasmExecChange: func() {
			// This callback is executed when wasm_exec.js content changes (e.g. mode switch)
			// We need to get the new content and update AssetMin
			// Note: We can't access h.wasmClient here directly if it's not assigned yet,
			// but since this callback is stored in config and called later, we need a way to get content.
			// However, we are inside AddSectionBUILD, so we can define the callback to use the handler variable
			// BUT we need to be careful about closure capture.
			// Actually, better approach: define the callback AFTER creating the instance?
			// No, Config is passed to New.
			// Alternative: The callback just signals "something changed", and we pull from handler?
			// Or we can use a closure that captures the *future* handler instance?
			// Let's try to capture the handler instance.
		},
	})

	h.wasmClient.SetAppRootDir(h.config.RootDir())
	h.wasmClient.CreateDefaultWasmFileClientIfNotExist()

	//ASSETS
	h.assetsHandler = assetmin.NewAssetMin(&assetmin.Config{
		OutputDir: filepath.Join(h.rootDir, h.config.WebPublicDir()),
		Logger:    assetsLogger,
		GetRuntimeInitializerJS: func() (string, error) {
			return h.wasmClient.JavascriptForInitializing()
		},
		AppName: h.frameworkName,
	})
	h.assetsHandler.SetWorkMode(assetmin.DiskMode)

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
		AppPort:  h.config.ServerPort(),
		Logger:   serverLogger,
		ExitChan: h.exitChan,
	})

	// BROWSER
	h.browser = devbrowser.New(h.config, h.tui, h.db, h.exitChan, browserLogger)

	// Wire up TinyWasm to AssetMin
	h.wasmClient.OnWasmExecChange = func() {
		// Notify AssetMin to refresh JS assets (this will pull the new initializer JS)
		h.assetsHandler.RefreshAsset(".js")
		wasmLogger("Refreshed script.js via AssetMin")

		// Reload the browser to apply changes
		if err := h.browser.Reload(); err != nil {
			wasmLogger("Error reloading browser:", err)
		} else {
			wasmLogger("Browser reload triggered")
		}
	}

	// WATCHER
	h.watcher = devwatch.New(&devwatch.WatchConfig{
		AppRootDir:         h.config.RootDir(),
		FilesEventHandlers: []devwatch.FilesEventHandlers{h.wasmClient, h.serverHandler, h.assetsHandler},
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
			uf = append(uf, h.wasmClient.UnobservedFiles()...)
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
	h.tui.AddHandler(h.wasmClient, time.Millisecond*500, colorPurpleMedium, sectionBuild)

}
