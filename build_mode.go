package app

import "github.com/tinywasm/server"

const StoreKeyBuildModeOnDisk = "build_mode_ondisk"

type BuildModeOnDisk struct {
	h *handler
}

func (b *BuildModeOnDisk) Name() string {
	return "BuildMode"
}

func (b *BuildModeOnDisk) Label() string {
	onDisk := false
	if val, err := b.h.db.Get(StoreKeyBuildModeOnDisk); err == nil && val == "true" {
		onDisk = true
	}

	if onDisk {
		return "BUILD: DISK"
	}
	return "BUILD: MEMORY"
}

func (b *BuildModeOnDisk) Execute(progress chan<- string) {
	onDisk := "false"
	if val, err := b.h.db.Get(StoreKeyBuildModeOnDisk); err == nil && val == "true" {
		onDisk = "true"
	}

	// Toggle
	if onDisk == "true" {
		onDisk = "false"
	} else {
		onDisk = "true"
	}

	b.h.db.Set(StoreKeyBuildModeOnDisk, onDisk)

	// Update handlers
	isDisk := (onDisk == "true")
	b.h.wasmClient.SetBuildOnDisk(isDisk)
	b.h.assetsHandler.SetBuildOnDisk(isDisk)
	b.h.serverHandler.SetBuildOnDisk(isDisk)

	if isDisk {
		progress <- "Switched to Build On Disk Mode"
	} else {
		progress <- "Switched to In-Memory Build Mode"
	}

	b.h.tui.RefreshUI()
}

func (h *handler) NewServerModeHandler() *server.ServerModeHandler {
	return server.NewServerModeHandler(h.serverHandler, h.db, h.tui)
}
