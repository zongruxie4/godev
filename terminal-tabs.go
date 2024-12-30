package godev

import (
	"sync"
)

// Terminal mantiene el estado de la aplicación
type Terminal struct {
	*terminalModel
	*terminalView
}

// NewTerminal crea una nueva instancia de Terminal
func NewTerminal() *Terminal {
	t := &Terminal{
		terminalModel: newTerminalModel(),
		terminalView:  newTerminalView(),
	}
	return t
}

func (t *Terminal) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	t.tea = t.terminalModel.startTeaProgram(t.terminalView)
	if _, err := t.tea.Run(); err != nil {
		t.MsgError("Error running program:", err)
	}
}
