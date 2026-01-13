package app

import (
	"github.com/tinywasm/wizard"
)

// AddSectionWIZARD registers the Wizard section in the TUI
func (h *handler) AddSectionWIZARD(onComplete func()) any {
	sectionWizard := h.tui.NewTabSection("WIZARD", "Project Initialization")

	// Get steps from GoNew and pass them to the wizard constructor.
	// We use the slice directly as it matches the new Wizard.New signature.
	steps := h.goNew.GetSteps()
	w := wizard.New(onComplete, steps...)

	h.tui.AddHandler(w, 0, "#00ADD8", sectionWizard) // Cyan color

	return sectionWizard
}
