package app

import (
	"net/http"
	"path/filepath"

	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/client"
	"github.com/tinywasm/devwatch"
	"github.com/tinywasm/server"
)

func (h *Handler) AddSectionBUILD() any {
	SectionBuild := h.Tui.NewTabSection("BUILD", "Building and Compiling")

	// CONFIG - only initialize if nil
	if h.Config == nil {
		h.Config = NewConfig(h.RootDir, nil)
	}

	// Register mode Handlers are no longer needed here as ServerHandler
	// is registered in InitBuildHandlers and implements HandlerEdit directly.

	return SectionBuild
}

// InitBuildHandlers initializes all build Handlers AFTER the project root is determined.
// This is called from OnProjectReady() to ensure paths are correct.
func (h *Handler) InitBuildHandlers() {
	// Skip if already initialized
	if h.WasmClient != nil {
		return
	}

	// 1. WASM Client - Core logic Handlers
	h.WasmClient = client.New(&client.Config{
		SourceDir: h.Config.CmdWebClientDir,
		OutputDir: h.Config.WebPublicDir,
		Database:  h.DB,
	})

	// Configurar AssetMin
	publicDir := filepath.Join(h.RootDir, h.Config.WebPublicDir())
	h.AssetsHandler = assetmin.NewAssetMin(&assetmin.Config{
		OutputDir: publicDir,
		GetSSRClientInitJS: func() (string, error) {
			return h.WasmClient.GetSSRClientInitJS()
		},
		AppName: h.FrameworkName,
	})

	// 3. SERVER
	h.ServerHandler = server.New(&server.Config{
		AppRootDir:                  h.RootDir,
		SourceDir:                   h.Config.CmdAppServerDir(),
		OutputDir:                   h.Config.DeployAppServerDir(),
		MainInputFile:               h.Config.ServerFileName(),
		Routes:                      []func(*http.ServeMux){h.AssetsHandler.RegisterRoutes, h.WasmClient.RegisterRoutes},
		ArgumentsForCompilingServer: func() []string { return []string{} },
		ArgumentsToRunServer: func() []string {
			args := []string{
				"-public-dir=" + filepath.Join(h.RootDir, h.Config.WebPublicDir()),
				"-port=" + h.Config.ServerPort(),
			}
			// APPENDED: Check dev mode
			if h.DevMode {
				args = append(args, "-dev")
			}
			return append(args, h.WasmClient.ArgumentsForServer()...)
		},
		AppPort:              h.Config.ServerPort(),
		DisableGlobalCleanup: TestMode,
		ExitChan:             h.ExitChan,
		Store:                h.DB,
		UI:                   h.Tui,
		OpenBrowser:          h.Browser.OpenBrowser, // Inject browser open callback
		OnExternalModeExecution: func(isExternal bool) {
			// Orchestrate client and assetmin to disk mode when using external server
			if h.WasmClient != nil {
				h.WasmClient.SetBuildOnDisk(isExternal, true)
			}
			if h.AssetsHandler != nil {
				h.AssetsHandler.SetExternalSSRCompiler(func() error { return nil }, isExternal)
			}
		},
		GitIgnoreAdd: h.GitHandler.GitIgnoreAdd,
	})

	// 4. BROWSER
	// Browser is already injected in Start()

	// 5. WATCHER
	h.Watcher = devwatch.New(&devwatch.WatchConfig{
		//AppRootDir: h.Config.RootDir, (Removed in favor of AddDirectoriesToWatch)
		FilesEventHandlers: []devwatch.FilesEventHandlers{
			h.GoModHandler,
			h.WasmClient,
			h.ServerHandler,
			h.AssetsHandler,
		},
		FolderEvents:  nil,
		BrowserReload: func() error { return h.Browser.Reload() },
		ExitChan:      h.ExitChan,
		UnobservedFiles: func() []string {
			uf := []string{
				".git",
				".gitignore",
				".vscode",
				".exe",
				".log",
				"_test.go",
			}
			uf = append(uf, h.AssetsHandler.UnobservedFiles()...)
			uf = append(uf, h.WasmClient.UnobservedFiles()...)
			uf = append(uf, h.ServerHandler.UnobservedFiles()...)
			return uf
		},
	})

	// 6. GO.MOD HANDLER
	// Use injected handler
	h.GoModHandler.SetLog(h.Watcher.Logger)
	h.GoModHandler.SetFolderWatcher(h.Watcher)

	// Add main project root to watcher
	h.Watcher.AddDirectoriesToWatch(h.Config.RootDir)

	// Add local replace modules to watcher automatically
	replaceEntries, err := h.GoModHandler.GetReplacePaths()
	if err == nil {
		var paths []string
		for _, entry := range replaceEntries {
			paths = append(paths, entry.LocalPath)
		}
		if len(paths) > 0 {
			h.Watcher.Logger("WATCH", "Watching local replacement modules:", paths)
			h.Watcher.AddDirectoriesToWatch(paths...)
		}
	} else {
		h.Watcher.Logger("Warning: failed to get replace paths:", err)
	}

	h.Watcher.SetShouldWatch(h.IsPartOfProject)

	// 6. Register Handlers with TUI for logging
	h.Tui.AddHandler(h.WasmClient, colorPurpleMedium, h.SectionBuild)
	h.Tui.AddHandler(h.ServerHandler, colorBlueMedium, h.SectionBuild)
	h.Tui.AddHandler(h.AssetsHandler, colorGreenMedium, h.SectionBuild)
	h.Tui.AddHandler(h.Watcher, colorYellowMedium, h.SectionBuild)
	h.Tui.AddHandler(h.Config, colorTealMedium, h.SectionBuild)
	h.Tui.AddHandler(h.Browser, colorPinkMedium, h.SectionBuild)

	// NOTE: GitHubAuth is registered in Start() BEFORE auth begins
	// to ensure it uses the TUI logger instead of file logger

	// 7. Wire up TinyWasm to AssetMin
	h.WasmClient.OnWasmExecChange = func() {
		h.AssetsHandler.RefreshAsset(".js")
		if err := h.Browser.Reload(); err != nil {
			h.WasmClient.Logger("Error reloading Browser:", err)
		}
	}

	// 8. Initialize deploy Handlers (depends on Watcher)
	h.InitDeployHandlers()
}
