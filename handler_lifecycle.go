package app

import (
	"sync"
)

const StoreKeyBuildModeOnDisk = "build_mode_ondisk"

// OnProjectReady is the centralized lifecycle method called when a project
// is fully initialized and ready to start its background services.
// It ensures all components have the correct flags and triggers the startup sequence.
func (h *Handler) OnProjectReady(wg *sync.WaitGroup) {
	// 0. Initialize build Handlers with correct paths (lazy initialization)
	h.InitBuildHandlers()

	// 1. Refresh component states
	// WasmClient needs to know it can now generate files and IDE configs
	h.WasmClient.SetAppRootDir(h.Config.RootDir)
	h.WasmClient.SetShouldCreateIDEConfig(h.IsInitializedProject)
	h.WasmClient.SetShouldGenerateDefaultFile(h.CanGenerateDefaultWasmClient)
	h.WasmClient.CreateDefaultWasmFileClientIfNotExist()

	// Ensure compilation happens (force recompile to load into memory)
	// This prevents 503 errors on subsequent runs where generation is skipped
	if err := h.WasmClient.RecompileMainWasm(); err != nil {
		h.WasmClient.Logger("Initial compilation failed:", err)
	}

	// DevWatch needs to know it should start watching files
	h.Watcher.SetShouldWatch(h.IsPartOfProject)

	// 2. Trigger background services sequence
	h.StartBackgroundServices(wg)
}

// StartBackgroundServices launches the server and Watcher in goroutines.
// It uses h.startOnce to guarantee services only start once per process.
func (h *Handler) StartBackgroundServices(wg *sync.WaitGroup) {
	h.startOnce.Do(func() {
		h.Tui.SetActiveTab(h.SectionBuild)

		// Start server (blocking, so run in goroutine)
		go func() {
			h.ServerHandler.StartServer(wg)
		}()

		// Start file Watcher (blocking, so run in goroutine)
		go h.Watcher.FileWatcherStart(wg)

	})
}
