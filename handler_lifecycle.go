package app

import (
	"sync"
	"time"
)

// onProjectReady is the centralized lifecycle method called when a project
// is fully initialized and ready to start its background services.
// It ensures all components have the correct flags and triggers the startup sequence.
func (h *handler) onProjectReady(wg *sync.WaitGroup) {
	// 0. Initialize build handlers with correct paths (lazy initialization)
	h.initBuildHandlers()

	// 1. Refresh component states
	// WasmClient needs to know it can now generate files and IDE configs
	h.wasmClient.SetAppRootDir(h.config.RootDir())
	h.wasmClient.SetShouldCreateIDEConfig(h.isInitializedProject)
	h.wasmClient.SetShouldGenerateDefaultFile(h.canGenerateDefaultWasmClient)
	h.wasmClient.CreateDefaultWasmFileClientIfNotExist()

	// DevWatch needs to know it should start watching files
	h.watcher.SetShouldWatch(h.isPartOfProject)

	// 2. Apply persisted work modes (Build Mode, External Server)
	h.applyPersistedWorkModes()

	// 3. Trigger background services sequence
	h.startBackgroundServices(wg)
}

// applyPersistedWorkModes reads work modes from the database and applies them to handlers.
func (h *handler) applyPersistedWorkModes() {
	if h.db == nil {
		return
	}

	// BUILD MODE (In-Memory vs On-Disk)
	if val, err := h.db.Get(StoreKeyBuildModeOnDisk); err == nil && val != "" {
		isDisk := (val == "true")
		h.wasmClient.SetBuildOnDisk(isDisk, true)
		h.assetsHandler.SetBuildOnDisk(isDisk)
		h.serverHandler.SetCompilationOnDisk(isDisk)
	} else {
		// Default to false (In-Memory) as requested
		h.wasmClient.SetBuildOnDisk(false, true)
		h.assetsHandler.SetBuildOnDisk(false)
		h.serverHandler.SetCompilationOnDisk(false)
	}

	// SERVER MODE (Internal vs External)
	// We use the key defined in the server package for consistency
	const StoreKeyExternalServer = "external_server"
	if val, err := h.db.Get(StoreKeyExternalServer); err == nil && val != "" {
		isExternal := (val == "true")
		_ = h.serverHandler.SetExternalServerMode(isExternal)
	} else {
		// Default to false (Internal) as requested
		_ = h.serverHandler.SetExternalServerMode(false)
	}
}

// startBackgroundServices launches the server and watcher in goroutines.
// It uses h.startOnce to guarantee services only start once per process.
func (h *handler) startBackgroundServices(wg *sync.WaitGroup) {
	h.startOnce.Do(func() {
		h.tui.SetActiveTab(h.sectionBuild)

		// Start server (blocking, so run in goroutine)
		go h.serverHandler.StartServer(wg)

		// Start file watcher (blocking, so run in goroutine)
		go h.watcher.FileWatcherStart(wg)

		// Auto-open browser (run in separate goroutine to not block main flow)
		if !TestMode {
			go func() {
				time.Sleep(100 * time.Millisecond)
				h.browser.AutoStart()
			}()
		}
	})
}
