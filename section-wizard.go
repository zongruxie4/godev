package app

import (
	"os"

	"github.com/tinywasm/context"
	"github.com/tinywasm/wizard"
)

// AddSectionWIZARD registers the Wizard section in the TUI
func (h *handler) AddSectionWIZARD(onComplete func()) any {
	// Add GoNew wizard steps
	h.goNew = h.goNew // Ensure it's reachable

	sectionWizard := h.tui.NewTabSection("WIZARD", "Project Initialization")

	w := wizard.New(func(ctx *context.Context) {
		// Extract project_dir from wizard context
		projectDir := ctx.Value("project_dir")
		if projectDir != "" {
			// Change working directory to new project
			if err := os.Chdir(projectDir); err == nil {
				// Update app config and rootDir
				h.config.SetRootDir(projectDir)
				h.rootDir = projectDir
			}
		}

		onComplete()
	}, h.goNew.Module()) // Passing it as a Module

	h.tui.AddHandler(w, 0, "#00ADD8", sectionWizard) // Cyan color

	return sectionWizard
}
