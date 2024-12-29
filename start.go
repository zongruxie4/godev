package godev

import (
	"sync"
)

type handler struct {
	terminal *Terminal
	program  *Program
}

func GodevStart() {
	h := &handler{
		terminal: NewTerminal(),
	}
	h.program = NewProgram(h.terminal)

	var wg sync.WaitGroup
	wg.Add(2)

	// Iniciar la terminal en una goroutine
	go h.terminal.Start(&wg)

	// Iniciar el programa
	go h.program.Start(&wg)
	wg.Wait()

}
