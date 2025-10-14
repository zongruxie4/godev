package godev

import (
	"time"

	"github.com/cdvelop/goflare"
)

func (h *handler) AddSectionDEPLOY() {

	sectionDeploy := h.tui.NewTabSection("DEPLOY", "Deploying Applications")

	// CLOUDFLARE (GOFLARE)
	h.deployCloudflare = goflare.New(&goflare.Config{
		AppRootDir:              h.config.rootDir,
		RelativeInputDirectory:  h.config.CmdEdgeWorkerDir(),
		RelativeOutputDirectory: h.config.DeployEdgeWorkerDir(),
		MainInputFile:           "main.go",
		Logger:                  h.tui.AddLogger("CLOUDFLARE", false, colorOrangeMedium, sectionDeploy),
		CompilingArguments:      nil,
		OutputWasmFileName:      "app.wasm",
	})

	h.tui.AddHandler(h.deployCloudflare, time.Millisecond*500, colorYellowLight, sectionDeploy)

}
