package app

import (
	"path/filepath"

	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/client"
	"github.com/tinywasm/devwatch"
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
		DevMode: h.DevMode, // Pass DevMode explicitly to prevent caching in development
	})

	// 3. SERVER
	h.Server = h.serverFactory()

	// Register routes directly
	h.Server.RegisterRoutes(h.AssetsHandler.RegisterRoutes)
	h.Server.RegisterRoutes(h.WasmClient.RegisterRoutes)

	// Wire server-specific callbacks via type assertion
	type externalModeSupport interface {
		SetOnExternalModeExecution(fn func(bool))
	}
	if srv, ok := h.Server.(externalModeSupport); ok {
		srv.SetOnExternalModeExecution(func(isExternal bool) {
			// Orchestrate client and assetmin to disk mode when using external server
			if h.WasmClient != nil {
				h.WasmClient.SetBuildOnDisk(isExternal, true)
			}
			if h.AssetsHandler != nil {
				h.AssetsHandler.SetExternalSSRCompiler(func() error { return nil }, isExternal)
			}
		})
	}

	// Configure server specific arguments
	type serverConfigurator interface {
		SetRunArgs(func() []string)
		SetCompileArgs(func() []string)
		SetAppRootDir(string)
		SetSourceDir(string)
		SetOutputDir(string)
		SetMainInputFile(string)
		SetPort(string)
		SetDisableGlobalCleanup(bool)
	}

	if srv, ok := h.Server.(serverConfigurator); ok {
		srv.SetAppRootDir(h.RootDir)
		srv.SetSourceDir(h.Config.CmdAppServerDir())
		srv.SetOutputDir(h.Config.DeployAppServerDir())
		srv.SetMainInputFile(h.Config.ServerFileName())
		srv.SetPort(h.Config.ServerPort())
		srv.SetDisableGlobalCleanup(TestMode)
		srv.SetCompileArgs(func() []string { return []string{"-p", "1"} })
		srv.SetRunArgs(func() []string {
			args := []string{
				"-public-dir=" + filepath.Join(h.RootDir, h.Config.WebPublicDir()),
				"-port=" + h.Config.ServerPort(),
			}
			// Check dev mode
			if h.DevMode {
				args = append(args, "-dev")
			}
			return append(args, h.WasmClient.ArgumentsForServer()...)
		})
	}

	// 4. BROWSER
	// Browser is already injected in Start()

	// 5. WATCHER
	h.Watcher = devwatch.New(&devwatch.WatchConfig{
		//AppRootDir: h.Config.RootDir, (Removed in favor of AddDirectoriesToWatch)
		FilesEventHandlers: []devwatch.FilesEventHandlers{
			h.GoModHandler,
			h.WasmClient,
			h.Server,
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
			uf = append(uf, h.Server.UnobservedFiles()...)
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
	h.Tui.AddHandler(h.Server, colorBlueMedium, h.SectionBuild)
	h.Tui.AddHandler(h.AssetsHandler, colorGreenMedium, h.SectionBuild)
	h.Tui.AddHandler(h.Watcher, colorYellowMedium, h.SectionBuild)
	h.Tui.AddHandler(h.Config, colorTealMedium, h.SectionBuild)
	h.Tui.AddHandler(h.Browser, colorPinkMedium, h.SectionBuild)

	// NOTE: GitHubAuth is registered in Start() BEFORE auth begins
	// to ensure it uses the TUI logger instead of file logger

	// 7. Wire up TinyWasm to AssetMin
	h.WasmClient.OnWasmExecChange = func() {
		h.AssetsHandler.RefreshAsset(".js")
		h.AssetsHandler.RefreshAsset(".html")

		// Restart server to pick up new mode arguments
		if err := h.Server.RestartServer(); err != nil {
			h.WasmClient.Logger("Error restarting Server:", err)
		}

		if err := h.Browser.Reload(); err != nil {
			h.WasmClient.Logger("Error reloading Browser:", err)
		}
	}

	// 8. Initialize deploy Handlers (depends on Watcher)
	h.InitDeployHandlers()
}
