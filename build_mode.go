package app

import "github.com/tinywasm/server"

const StoreKeyBuildModeOnDisk = "build_mode_ondisk"

type BuildModeOnDisk struct {
	h   *handler
	log func(message ...any)
}

func (b *BuildModeOnDisk) SetLog(f func(message ...any)) {
	b.log = f
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
		return "BUILD FILES: ONDISK"
	}
	return "BUILD FILES: IN-MEMORY"
}

func (b *BuildModeOnDisk) Execute() {
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
	b.h.wasmClient.SetBuildOnDisk(isDisk, true)
	b.h.assetsHandler.SetBuildOnDisk(isDisk)
	b.h.serverHandler.SetCompilationOnDisk(isDisk)

	if b.log != nil {
		if isDisk {
			b.log("Switched to Build Files: ONDISK Mode")
		} else {
			b.log("Switched to Build Files: IN-MEMORY Mode")
		}
	}

	b.h.tui.RefreshUI()
}

func (h *handler) NewServerModeHandler() *server.ServerModeHandler {
	return server.NewServerModeHandler(h.serverHandler, h.db, h.tui)
}
