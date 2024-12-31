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

	go h.FileWatcherStart(&wg)

	// Iniciar el programa
	go h.program.Start(&wg)
	// Esperar a que todas las goroutines terminen
	go func() {
		<-exitChan
		close(exitChan)
		wg.Wait()
	}()

	wg.Wait()

}
