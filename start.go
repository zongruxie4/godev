package godev

import (
	"sync"

	"github.com/fsnotify/fsnotify"
)

type handler struct {
	terminal *Terminal
	watcher  *fsnotify.Watcher
	program  *Program
}

// Canal global para señalizar el cierre
var exitChan = make(chan bool)

// func GodevStart() {
// 	h := &handler{
// 		terminal: NewTerminal(),
// 	}
// 	h.program = NewProgram(h.terminal)

// 	if watcher, err := fsnotify.NewWatcher(); err != nil {
// 		configErrors = append(configErrors, err)
// 	} else {
// 		h.watcher = watcher
// 	}

// 	var wg sync.WaitGroup
// 	wg.Add(2) // Reducimos a 2 porque la terminal se maneja con Bubbletea

// 	// mostrar errores de configuración como warning
// 	if len(configErrors) != 0 {
// 		for _, err := range configErrors {
// 			h.terminal.MsgWarning(err)
// 		}
// 	}

// 	// Iniciar el watcher
// 	go h.FileWatcherStart(&wg)

// 	// Iniciar el programa
// 	go h.program.Start(&wg)

// 	// La terminal ya maneja el exitChan con Bubbletea

// 	// Esperar a que las goroutines terminen
// 	wg.Wait()
// }

func GodevStart() {

	h := &handler{
		terminal: NewTerminal(),
	}
	h.program = NewProgram(h.terminal)

	if watcher, err := fsnotify.NewWatcher(); err != nil {
		configErrors = append(configErrors, err)
	} else {
		h.watcher = watcher
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// Iniciar la terminal en una goroutine
	go h.terminal.Start(&wg)

	// mostrar errores de configuración como warning
	if len(configErrors) != 0 {
		for _, err := range configErrors {
			h.terminal.MsgWarning(err)
		}
	}

	// Iniciar el programa
	go h.program.Start(&wg)

	go h.FileWatcherStart(&wg)

	// Esperar a que todas las goroutines terminen

	wg.Wait()

	for {

		select {

		case <-exitChan:
			// Detener la terminal
			// h.terminal.tea.Quit()
			// os.Exit(0)
			return

		}

	}

}
