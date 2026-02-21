package app

import (
	"path/filepath"

	"github.com/tinywasm/context"
	"github.com/tinywasm/deploy"
	"github.com/tinywasm/goflare"
	"github.com/tinywasm/wizard"
)

func (h *Handler) AddSectionDEPLOY() any {
	SectionDeploy := h.Tui.NewTabSection("DEPLOY", "Deploying Applications")
	h.SectionDeploy = SectionDeploy
	return SectionDeploy
}

// InitDeployHandlers initializes deploy handlers after build handlers are ready.
// Called from InitBuildHandlers to ensure Watcher exists.
func (h *Handler) InitDeployHandlers() {
	// goflare: pure WASM compiler for Cloudflare Workers/Pages
	h.DeployCloudflare = goflare.New(&goflare.Config{
		AppRootDir:              h.Config.RootDir,
		RelativeInputDirectory:  h.Config.CmdEdgeWorkerDir,
		RelativeOutputDirectory: h.Config.DeployEdgeWorkerDir,
		MainInputFile:           "main.go",
		CompilingArguments:      nil,
		OutputWasmFileName:      "app.wasm",
	})
	h.Tui.AddHandler(h.DeployCloudflare, colorYellowLight, h.SectionDeploy)
	h.Watcher.AddFilesEventHandlers(h.DeployCloudflare)

	// deploy: single orchestrator for all deploy methods (cloudflare/webhook/ssh)
	// h.DB satisfies deploy.Store directly (kvdb.KVStore has Get/Set flat keys)
	d := &deploy.Deploy{
		Store:      h.DB,
		Process:    deploy.NewProcessManager(),
		Downloader: deploy.NewDownloader(),
		Checker:    deploy.NewChecker(),
		ConfigPath: filepath.Join(h.RootDir, "deploy.yaml"),
	}
	d.SetLog(h.Logger)

	if d.IsConfigured() {
		h.Tui.AddHandler(d, colorOrangeLight, h.SectionDeploy)
	} else {
		h.initDeployWizard(d)
	}
}

func (h *Handler) initDeployWizard(d *deploy.Deploy) {
	sectionSetup := h.Tui.NewTabSection("DEPLOY SETUP", "Deployment Configuration")

	w := wizard.New(func(ctx *context.Context) {
		h.Tui.SetActiveTab(h.SectionDeploy)
		h.Tui.AddHandler(d, colorOrangeLight, h.SectionDeploy)
	}, d)

	h.Tui.AddHandler(w, colorOrangeLight, sectionSetup)
	h.Tui.SetActiveTab(sectionSetup)
}
