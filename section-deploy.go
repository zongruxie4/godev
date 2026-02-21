package app

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/goflare"
	"github.com/tinywasm/wizard"
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

	h.DeployCloudflare.SetKeys(h.Keys)

	h.Tui.AddHandler(h.DeployCloudflare, colorYellowLight, h.SectionDeploy)
	h.Watcher.AddFilesEventHandlers(h.DeployCloudflare)

	cfAuth := h.DeployCloudflare.Auth()
	if !cfAuth.IsConfigured() {
		h.initCFAuthWizard(cfAuth)
	}
}

func (h *Handler) initCFAuthWizard(auth *goflare.Auth) {
	SectionWizardDeploy := h.Tui.NewTabSection("DEPLOY SETUP", "Cloudflare Configuration")

	w := wizard.New(func(ctx *context.Context) {
		h.Tui.SetActiveTab(h.SectionDeploy)
	}, auth)

	h.Tui.AddHandler(w, colorYellowLight, SectionWizardDeploy)
	h.Tui.SetActiveTab(SectionWizardDeploy)
}
