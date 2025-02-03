package godev

import (
	"sync"

	"github.com/fsnotify/fsnotify"
)

type handler struct {
	terminal *Terminal
	watcher  *fsnotify.Watcher
	program  *Program
	browser  *Browser
}

// Canal global para señalizar el cierre
var exitChan = make(chan bool)

func GodevStart() {

	bws := NewBrowser()

	h := &handler{}

	h.browser = bws
	h.terminal = NewTerminal(bws)
	h.program = NewProgram(h.terminal)

	if watcher, err := fsnotify.NewWatcher(); err != nil {
		configErrors = append(configErrors, err)
	} else {
		h.watcher = watcher
		defer h.watcher.Close()
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// Iniciar la terminal en una goroutine
	go h.terminal.Start(&wg)

	// Mostrar errores de configuración como warning
	if len(configErrors) != 0 {
		for _, err := range configErrors {
			h.terminal.MsgWarning(err)
		}
	}

	// Iniciar el programa
	go h.program.Start(&wg)

	// Iniciar el watcher de archivos
	go h.FileWatcherStart(&wg)

	// Esperar a que todas las goroutines terminen
	wg.Wait()
}
