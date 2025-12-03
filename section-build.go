package golite

import (
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
	wasmLogger := h.tui.AddLogger("WASM", true, colorPurpleMedium, sectionBuild)
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
		SourceDir:                   h.config.CmdAppServerDir(),
		OutputDir:                   h.config.DeployAppServerDir(),
		MainInputFile:               h.config.ServerFileName(),
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

	//WASM
	h.wasmHandler = tinywasm.New(&tinywasm.Config{
		AppRootDir:              h.rootDir,
		SourceDir:               h.config.CmdWebClientDir(),
		MainInputFile:           h.config.ClientFileName(),
		OutputDir:               h.config.WebPublicDir(),
		WasmExecJsOutputDir:     filepath.Join(h.config.JsDir()),
		Logger:                  wasmLogger,
		Store:                   h.db,
		DisableWasmExecJsOutput: true, // AssetMin handles writing
		OnWasmExecChange: func() {
			// This callback is executed when wasm_exec.js content changes (e.g. mode switch)
			// We need to get the new content and update AssetMin
			// Note: We can't access h.wasmHandler here directly if it's not assigned yet,
			// but since this callback is stored in config and called later, we need a way to get content.
			// However, we are inside AddSectionBUILD, so we can define the callback to use the handler variable
			// BUT we need to be careful about closure capture.
			// Actually, better approach: define the callback AFTER creating the instance?
			// No, Config is passed to New.
			// Alternative: The callback just signals "something changed", and we pull from handler?
			// Or we can use a closure that captures the *future* handler instance?
			// Let's try to capture the handler instance.
		},
	}).CreateDefaultWasmFileClientIfNotExist()

	//ASSETS
	h.assetsHandler = NewAssetMin(&Config{
		ThemeFolder: func() string {
			return filepath.Join(h.rootDir, h.config.WebUIDir())
		},
		OutputDir: func() string {
			return filepath.Join(h.rootDir, h.config.WebPublicDir())
		},
		Logger: assetsLogger,
		GetRuntimeInitializerJS: func() (string, error) {
			return h.wasmHandler.JavascriptForInitializing()
		},
		AppName: h.frameworkName,
	}).CreateDefaultIndexHtmlIfNotExist().
		CreateDefaultCssIfNotExist().
		CreateDefaultJsIfNotExist().
		CreateDefaultFaviconIfNotExist()

	// BROWSER
	h.browser = devbrowser.New(h.config, h.tui, h.db, h.exitChan, browserLogger)

	// Wire up TinyWasm to AssetMin
	h.wasmHandler.OnWasmExecChange = func() {
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
