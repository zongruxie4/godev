package app

import (
	"path/filepath"
)

type WizardDeps interface {
	SetProjectName(name string)
	SetRootDir(path string)
	GetRootDir() string
	SetActiveTab(section any)
}

// Wizard implements HandlerInteractive for project setup
type Wizard struct {
	deps           WizardDeps
	log            func(message ...any)
	label          string
	currentValue   string
	waitingForUser bool
	wizardSection  any // Store section reference for removal

	step       int
	onComplete func() // Callback to execute after completion
}

func NewWizard(deps WizardDeps, onComplete func()) *Wizard {
	return &Wizard{
		deps:           deps,
		log:            func(...any) {},
		label:          "Project Name",
		waitingForUser: true, // Auto-activate input on start
		onComplete:     onComplete,
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

	case 1: // Set project location
		if newValue == "" {
			w.log("Error: project location is required")
			return
		}
		w.deps.SetRootDir(newValue) // FIX: Root directory must be updated early
		w.log("Project location: " + newValue)

		w.label = ""
		w.step = 2
		w.waitingForUser = false // Done waiting
		w.Change("")             // Trigger step 2 transition
	case 2: // Accept suggestions
		if w.onComplete != nil {
			w.onComplete()
			w.onComplete = nil // Clear to prevent re-execution
		}
		w.step = 3 // Final state: Completed
		w.waitingForUser = false

	case 3:
		// Silent state: do nothing. This prevents redundant logs during navigation.
	}
}

// Loggable implementation
func (w *Wizard) SetLog(f func(message ...any)) { w.log = f }

// StreamingLoggable implementation - show all logs instead of overwriting
func (w *Wizard) AlwaysShowAllLogs() bool { return true }

// Cancelable implementation - handle ESC gracefully
func (w *Wizard) Cancel() {
	w.waitingForUser = false
	w.log("Setup cancelled")
}

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

func (h *handler) SetActiveTab(section any) {
	h.tui.SetActiveTab(section)
}

// AddSectionWIZARD registers the Wizard section in the TUI
func (h *handler) AddSectionWIZARD(onComplete func()) any {
	sectionWizard := h.tui.NewTabSection("WIZARD", "Project Initialization")

	wizard := NewWizard(h, onComplete)
	wizard.SetSection(sectionWizard)

	h.tui.AddHandler(wizard, 0, "#00ADD8", sectionWizard) // Cyan color

	return sectionWizard
}
