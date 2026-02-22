package app

import (
	"path/filepath"

	"github.com/tinywasm/context"
	"github.com/tinywasm/deploy"
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
	d := deploy.NewDaemon(&deploy.DaemonConfig{
		AppRootDir:          h.Config.RootDir,
		CmdEdgeWorkerDir:    h.Config.CmdEdgeWorkerDir(),
		DeployEdgeWorkerDir: h.Config.DeployEdgeWorkerDir(),
		OutputWasmFileName:  "app.wasm",
		DeployConfigPath:    filepath.Join(h.RootDir, "deploy.yaml"),
		Store:               h.DB,
	})
	d.SetLog(h.Logger)
	h.DeployManager = d

	h.Tui.AddHandler(d.EdgeWorker(), colorYellowLight, h.SectionDeploy)
	h.Watcher.AddFilesEventHandlers(d)

	if d.IsConfigured() {
		h.Tui.AddHandler(d.Puller(), colorOrangeLight, h.SectionDeploy)
	} else {
		h.initDeployWizard(d)
	}
}

func (h *Handler) initDeployWizard(d *deploy.Daemon) {
	w := wizard.New(func(ctx *context.Context) {
		h.Tui.AddHandler(d.Puller(), colorOrangeLight, h.SectionDeploy)
	}, d)

	h.Tui.AddHandler(w, colorOrangeLight, h.SectionDeploy)
	h.Tui.SetActiveTab(h.SectionDeploy)
}
