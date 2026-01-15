package app

import (
	"os"

	"github.com/tinywasm/context"
	"github.com/tinywasm/wizard"
)

// AddSectionWIZARD registers the Wizard section in the TUI
func (h *handler) AddSectionWIZARD(onComplete func()) any {
	// Add GoNew wizard steps

	sectionWizard := h.tui.NewTabSection("WIZARD", "Project Initialization")

	w := wizard.New(func(ctx *context.Context) {
		// Extract project_dir from wizard context
		projectDir := ctx.Value("project_dir")
		if projectDir != "" {
			// Change working directory to new project
			if err := os.Chdir(projectDir); err == nil {
				// Update all path-dependent components
				h.config.SetRootDir(projectDir)
				h.rootDir = projectDir
				h.gitHandler.SetRootDir(projectDir)
				h.goHandler.SetRootDir(projectDir)
			}
		}

		onComplete()
	}, h.goNew) // Passing it directly (transparent casting)

	h.tui.AddHandler(w, 0, "#00ADD8", sectionWizard) // Cyan color

	return sectionWizard
}
