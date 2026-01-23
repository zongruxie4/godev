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

	// DevWatch needs to know it should start watching files
	h.Watcher.SetShouldWatch(h.IsPartOfProject)

	// 2. Apply persisted work modes (Build Mode, External Server)
	h.ApplyPersistedWorkModes()

	// 3. Trigger background services sequence
	h.StartBackgroundServices(wg)
}

// ApplyPersistedWorkModes reads work modes from the database and applies them to Handlers.
func (h *Handler) ApplyPersistedWorkModes() {
	if h.DB == nil {
		return
	}

	// BUILD MODE (In-Memory vs On-Disk)
	if val, err := h.DB.Get(StoreKeyBuildModeOnDisk); err == nil && val != "" {
		isDisk := (val == "true")
		h.WasmClient.SetBuildOnDisk(isDisk, true)
		h.AssetsHandler.SetBuildOnDisk(isDisk)
	} else {
		// Default to false (In-Memory) as requested
		h.WasmClient.SetBuildOnDisk(false, true)
		h.AssetsHandler.SetBuildOnDisk(false)
	}
}

// StartBackgroundServices launches the server and Watcher in goroutines.
// It uses h.startOnce to guarantee services only start once per process.
func (h *Handler) StartBackgroundServices(wg *sync.WaitGroup) {
	h.startOnce.Do(func() {
		h.Tui.SetActiveTab(h.SectionBuild)

		// Start server (blocking, so run in goroutine)
		go h.ServerHandler.StartServer(wg)

		// Start file Watcher (blocking, so run in goroutine)
		go h.Watcher.FileWatcherStart(wg)

	})
}
