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
	sectionBuild := h.tui.NewTabSection("BUILD", "Building and Compiling")

	// CONFIG - only initialize if nil
	if h.config == nil {
		h.config = NewConfig(h.rootDir, nil)
	}

	// Register mode handlers (these don't depend on rootDir paths)
	h.tui.AddHandler(&BuildModeOnDisk{h: h}, time.Millisecond*500, colorTealMedium, sectionBuild)
	h.tui.AddHandler(h.NewServerModeHandler(), time.Millisecond*500, colorBlueMedium, sectionBuild)

	return sectionBuild
}

// initBuildHandlers initializes all build handlers AFTER the project root is determined.
// This is called from onProjectReady() to ensure paths are correct.
func (h *handler) initBuildHandlers() {
	// Skip if already initialized
	if h.wasmClient != nil {
		return
	}

	// 1. WASM Client - Core logic handlers
	h.wasmClient = client.New(&client.Config{
		SourceDir: h.config.CmdWebClientDir,
		OutputDir: h.config.WebPublicDir,
		Database:  h.db,
	})

	// 2. ASSETS
	h.assetsHandler = assetmin.NewAssetMin(&assetmin.Config{
		OutputDir: filepath.Join(h.rootDir, h.config.WebPublicDir()),
		GetRuntimeInitializerJS: func() (string, error) {
			return h.wasmClient.JavascriptForInitializing()
		},
		AppName: h.frameworkName,
	})

	// 3. SERVER
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

	// 4. BROWSER
	h.browser = devbrowser.New(h.config, h.tui, h.db, h.exitChan)

	// Create browser reload wrapper that supports test hooks
	browserReloadFunc := func() error {
		if f := GetInitialBrowserReloadFunc(); f != nil {
			return f()
		}
		return h.browser.Reload()
	}

	// 5. WATCHER
	h.watcher = devwatch.New(&devwatch.WatchConfig{
		AppRootDir:         h.config.RootDir(),
		FilesEventHandlers: []devwatch.FilesEventHandlers{h.wasmClient, h.serverHandler, h.assetsHandler},
		FolderEvents:       nil,
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

	// 6. Register handlers with TUI for logging
	h.tui.AddHandler(h.wasmClient, 0, colorPurpleMedium, h.sectionBuild)
	h.tui.AddHandler(h.serverHandler, 0, colorBlueMedium, h.sectionBuild)
	h.tui.AddHandler(h.assetsHandler, 0, colorGreenMedium, h.sectionBuild)
	h.tui.AddHandler(h.watcher, 0, colorYellowMedium, h.sectionBuild)
	h.tui.AddHandler(h.config, 0, colorTealMedium, h.sectionBuild)
	h.tui.AddHandler(h.browser, 0, colorPinkMedium, h.sectionBuild)

	// 7. Wire up TinyWasm to AssetMin
	h.wasmClient.OnWasmExecChange = func() {
		h.assetsHandler.RefreshAsset(".js")
		if err := h.browser.Reload(); err != nil {
			h.wasmClient.Logger("Error reloading browser:", err)
		}
	}

	// 8. Initialize deploy handlers (depends on watcher)
	h.initDeployHandlers()
}
