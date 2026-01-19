package app

import (
	"time"

	"github.com/tinywasm/goflare"
)

func (h *Handler) AddSectionDEPLOY() any {
	SectionDeploy := h.Tui.NewTabSection("DEPLOY", "Deploying Applications")

	// Store section for later use in InitDeployHandlers
	h.SectionDeploy = SectionDeploy

	return SectionDeploy
}

// InitDeployHandlers initializes deploy Handlers after build Handlers are ready.
// Called from InitBuildHandlers to ensure Watcher exists.
func (h *Handler) InitDeployHandlers() {
	// CLOUDFLARE (GOFLARE)
	h.DeployCloudflare = goflare.New(&goflare.Config{
		AppRootDir:              h.Config.RootDir,
		RelativeInputDirectory:  h.Config.CmdEdgeWorkerDir,
		RelativeOutputDirectory: h.Config.DeployEdgeWorkerDir,
		MainInputFile:           "main.go",
		CompilingArguments:      nil,
		OutputWasmFileName:      "app.wasm",
	})

	h.Tui.AddHandler(h.DeployCloudflare, time.Millisecond*500, colorYellowLight, h.SectionDeploy)
	h.Watcher.AddFilesEventHandlers(h.DeployCloudflare)
}
