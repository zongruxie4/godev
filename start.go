package godev

import (
	"sync"

	"github.com/fsnotify/fsnotify"
)

type handler struct {
	ch      *ConfigHandler
	tui     *TextUserInterface
	watcher *fsnotify.Watcher
	program *Program
	browser *Browser
}

// Canal global para señalizar el cierre
var exitChan = make(chan bool)

func GodevStart() {

	h := &handler{}

	h.NewConfig()

	h.NewTextUserInterface()
	h.NewProgram()

	h.NewWatcher()
	defer h.watcher.Close()

	h.NewBrowser()
	// if watcher, err := fsnotify.NewWatcher(); err != nil {
	// 	configErrors = append(configErrors, err)
	// } else {
	// 	h.watcher = watcher
	// 	defer h.watcher.Close()
	// }
	var wg sync.WaitGroup
	wg.Add(3)

	// Iniciar la tui en una goroutine
	go h.Start(&wg)

	// Mostrar errores de configuración como warning
	if len(h.ch.configErrors) != 0 {
		for _, err := range h.ch.configErrors {
			h.tui.MsgWarning(err)
		}
	}

	// Iniciar el programa
	go h.ProgramStart(&wg)

	// Iniciar el watcher de archivos
	go h.FileWatcherStart(&wg)

	// Esperar a que todas las goroutines terminen
	wg.Wait()
}
