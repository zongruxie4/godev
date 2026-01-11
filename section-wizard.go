package app

import (
	"path/filepath"
)

type WizardDeps interface {
	SetProjectName(name string)
	SetRootDir(path string)
	GetRootDir() string
	RemoveSection(section any)
}

// Wizard implements HandlerInteractive for project setup
type Wizard struct {
	deps           WizardDeps
	log            func(message ...any)
	label          string
	currentValue   string
	waitingForUser bool
	wizardSection  any // Store section reference for removal

	step int
}

func NewWizard(deps WizardDeps) *Wizard {
	return &Wizard{
		deps:           deps,
		log:            func(...any) {},
		label:          "Project Name",
		waitingForUser: true, // Auto-activate input on start
	}
}

// HandlerInteractive implementation

func (w *Wizard) Name() string { return "ProjectSetup" }
func (w *Wizard) Label() string {
	return w.label
}
func (w *Wizard) Value() string        { return w.currentValue }
func (w *Wizard) WaitingForUser() bool { return w.waitingForUser }

func (w *Wizard) Change(newValue string) {
	switch w.step {
	case 0: // Step 0: Get project name
		if newValue == "" {
			w.log("Please enter a project name")
			return
		}
		w.deps.SetProjectName(newValue)
		w.log("Project name: " + newValue)

		// Prepare step 1: suggest location
		rootDir := w.deps.GetRootDir()
		parentFolder := filepath.Base(rootDir)
		w.currentValue = parentFolder + "/" + newValue

		w.label = "Project Location"
		w.step = 1
		w.waitingForUser = true // Wait for location input

	case 1: // Step 1: Get project location
		if newValue == "" {
			w.log("Please enter a project location")
			return
		}
		w.log("Project location: " + newValue)

		w.label = ""
		w.step = 2
		w.waitingForUser = false // Done waiting
		w.Change("")             // Trigger step 2 immediately

	case 2: // Step 2: Close wizard
		w.deps.RemoveSection(w.wizardSection)
	}
}

// Loggable implementation
func (w *Wizard) SetLog(f func(message ...any)) { w.log = f }

// SetSection stores the section reference for later removal
func (w *Wizard) SetSection(section any) { w.wizardSection = section }

// handler implementation of WizardDeps
// This allows the Wizard to interact with the application state

func (h *handler) SetProjectName(name string) {
	h.config.SetAppName(name)
}

func (h *handler) SetRootDir(path string) {
	h.rootDir = path
	h.config.SetRootDir(path)
}

func (h *handler) GetRootDir() string {
	return h.rootDir
}

func (h *handler) RemoveSection(section any) {
	h.tui.RemoveTabSection(section)
}

// AddSectionWIZARD registers the Wizard section in the TUI
func (h *handler) AddSectionWIZARD() any {
	sectionWizard := h.tui.NewTabSection("WIZARD", "Project Initialization")

	wizard := NewWizard(h)
	wizard.SetSection(sectionWizard)

	h.tui.AddHandler(wizard, 0, "#00ADD8", sectionWizard) // Cyan color

	return sectionWizard
}
