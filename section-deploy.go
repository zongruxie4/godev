package app

import (
	"time"

	"github.com/tinywasm/goflare"
)

func (h *handler) AddSectionDEPLOY() {
	sectionDeploy := h.tui.NewTabSection("DEPLOY", "Deploying Applications")

	// Store section for later use in initDeployHandlers
	h.sectionDeploy = sectionDeploy
}

// initDeployHandlers initializes deploy handlers after build handlers are ready.
// Called from initBuildHandlers to ensure watcher exists.
func (h *handler) initDeployHandlers() {
	// CLOUDFLARE (GOFLARE)
	h.deployCloudflare = goflare.New(&goflare.Config{
		AppRootDir:              h.config.rootDir,
		RelativeInputDirectory:  h.config.CmdEdgeWorkerDir,
		RelativeOutputDirectory: h.config.DeployEdgeWorkerDir,
		MainInputFile:           "main.go",
		CompilingArguments:      nil,
		OutputWasmFileName:      "app.wasm",
	})

	h.tui.AddHandler(h.deployCloudflare, time.Millisecond*500, colorYellowLight, h.sectionDeploy)
	h.watcher.AddFilesEventHandlers(h.deployCloudflare)
}
