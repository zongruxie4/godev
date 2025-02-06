package godev

import (
	"sync"

	"github.com/fsnotify/fsnotify"
)

type handler struct {
	terminal *TextUserInterface
	watcher  *fsnotify.Watcher
	program  *Program
	browser  *Browser
}

// Canal global para señalizar el cierre
var exitChan = make(chan bool)

func GodevStart() {

	h := &handler{}

	h.NewBrowser()
	h.NewTextUserInterface()
	h.NewProgram()

	h.NewWatcher()
	defer h.watcher.Close()

	// if watcher, err := fsnotify.NewWatcher(); err != nil {
	// 	configErrors = append(configErrors, err)
	// } else {
	// 	h.watcher = watcher
	// 	defer h.watcher.Close()
	// }
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
	go h.ProgramStart(&wg)

	// Iniciar el watcher de archivos
	go h.FileWatcherStart(&wg)

	// Esperar a que todas las goroutines terminen
	wg.Wait()
}
