package godev

import (
	"path"
	"time"

	"github.com/cdvelop/goflare"
)

func (h *handler) AddSectionDEPLOY() {

	sectionDeploy := h.tui.NewTabSection("DEPLOY", "Deploying Applications")

	// CLOUDFLARE (GOFLARE)
	h.deployCloudflare = goflare.New(&goflare.Config{
		AppRootDir:              h.rootDir,
		RelativeInputDirectory:  path.Join(h.config.GetWebFilesFolder(), "edgeworker"),
		RelativeOutputDirectory: path.Join(h.config.GetWebFilesFolder(), "edgeworker", "deploy"),
	})

	sectionDeploy.AddEditHandler(h.deployCloudflare, time.Millisecond*500, colorYellowLight)

}
