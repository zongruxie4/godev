package godev

import (
	"path"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type handler struct {
	ch             *ConfigHandler
	tui            *TextUserInterface
	assetsCompiler *AssetsCompiler
	wasmCompiler   *WasmCompiler
	watcher        *fsnotify.Watcher
	program        *Program
	browser        *Browser
	exitChan       chan bool // Canal global para señalizar el cierre
}

func GodevStart() {

	h := &handler{
		exitChan: make(chan bool),
	}

	h.NewConfig()
	h.NewTextUserInterface()
	h.NewProgram()
	h.AddCompilers()
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
			h.tui.PrintWarning(err)
		}
	}

	// Iniciar el programa
	go h.ProgramStart(&wg)

	// Iniciar el watcher de archivos
	go h.FileWatcherStart(&wg)

	// Esperar a que todas las goroutines terminen
	wg.Wait()
}

func (h *handler) AddCompilers() {

	const outputFromPublicWebFiles = "public"

	h.wasmCompiler = NewWasmCompiler(&WasmConfig{
		BuildDirectory: func() string {
			return path.Join(h.ch.config.OutputDir, outputFromPublicWebFiles, "wasm")
		},
		Print: h.tui.Print,
	})

	h.assetsCompiler = NewAssetsCompiler(&AssetsConfig{
		BuildDirectory: func() string {
			return path.Join(h.ch.config.OutputDir)
		},
		Print:                  h.tui.Print,
		WasmProjectTinyGoJsUse: h.wasmCompiler.WasmProjectTinyGoJsUse,
	})

}
