package app

import (
	"os"

	"github.com/tinywasm/context"
	"github.com/tinywasm/wizard"
)

// AddSectionWIZARD registers the Wizard section in the TUI
func (h *Handler) AddSectionWIZARD(onSuccess func()) any {
	// Add GoNew wizard steps

	sectionWizard := h.Tui.NewTabSection("WIZARD", "Project Initialization")

	w := wizard.New(func(ctx *context.Context) {
		// Extract project_dir from wizard context
		projectDir := ctx.Value("project_dir")
		if projectDir != "" {
			// Change working directory to new project
			if err := os.Chdir(projectDir); err == nil {
				// Update all path-dependent components
				h.Config.SetRootDir(projectDir)
				h.RootDir = projectDir
				h.GitHandler.SetRootDir(projectDir)
				h.GoHandler.SetRootDir(projectDir)
			}
		}

		onSuccess()
	}, h.GoNew) // Passing it directly (transparent casting)

	h.Tui.AddHandler(w, "#00ADD8", sectionWizard) // Cyan color

	return sectionWizard
}
