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
	goCompiler     *GoCompiler
	browser        *Browser
	exitChan       chan bool // Canal global para señalizar el cierre
}

func GodevStart() {

	h := &handler{
		exitChan: make(chan bool),
	}

	h.NewConfig()
	h.NewTextUserInterface()
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
	go h.goCompiler.Start(&wg)

	// Iniciar el watcher de archivos
	go h.FileWatcherStart(&wg)

	// Esperar a que todas las goroutines terminen
	wg.Wait()
}

func (h *handler) AddCompilers() {

	const publicWebFolder = "public"

	//GO
	h.goCompiler = NewGoCompiler(&GoCompilerConfig{
		MainFilePath: func() string {
			return path.Join(h.ch.config.WebFilesFolder, "main.server.go")
		},
		OutPathAppName: func() string {
			return path.Join(h.ch.config.WebFilesFolder, h.ch.config.AppName)
		},
		RunArguments: func() []string {
			return []string{}
		},
		Print:    h.tui.Print,
		Writer:   h,
		ExitChan: h.exitChan,
	})

	//WASM
	h.wasmCompiler = NewWasmCompiler(&WasmConfig{
		WebFilesFolder: func() (string, string) {
			return h.ch.config.WebFilesFolder, publicWebFolder
		},
		Print: h.tui.Print,
	})

	//ASSETS
	h.assetsCompiler = NewAssetsCompiler(&AssetsConfig{
		WebFilesFolder: func() string {
			return path.Join(h.ch.config.WebFilesFolder, publicWebFolder)
		},
		Print:                  h.tui.Print,
		WasmProjectTinyGoJsUse: h.wasmCompiler.WasmProjectTinyGoJsUse,
	})

}
