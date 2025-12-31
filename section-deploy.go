package app

import (
	"time"

	"github.com/tinywasm/goflare"
)

func (h *handler) AddSectionDEPLOY() {

	sectionDeploy := h.tui.NewTabSection("DEPLOY", "Deploying Applications")

	// CLOUDFLARE (GOFLARE)
	h.deployCloudflare = goflare.New(&goflare.Config{
		AppRootDir:              h.config.rootDir,
		RelativeInputDirectory:  h.config.CmdEdgeWorkerDir(),
		RelativeOutputDirectory: h.config.DeployEdgeWorkerDir(),
		MainInputFile:           "main.go",
		CompilingArguments:      nil,
		OutputWasmFileName:      "app.wasm",
	})

	h.tui.AddHandler(h.deployCloudflare, time.Millisecond*500, colorYellowLight, sectionDeploy)

	h.watcher.AddFilesEventHandlers(h.deployCloudflare)

}
