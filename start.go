package godev

import (
	"sync"
)

type handler struct {
	ch            *ConfigHandler
	tui           *TextUserInterface
	serverHandler *ServerHandler
	assetsHandler *AssetsHandler
	wasmHandler   *WasmHandler
	watcher       *WatchHandler
	browser       *Browser
	exitChan      chan bool // Canal global para señalizar el cierre
}

func GodevStart() {

	h := &handler{
		exitChan: make(chan bool),
	}

	h.NewConfig()
	h.NewTextUserInterface()
	h.AddHandlers()

	h.NewBrowser()

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

	// Iniciar servidor
	go h.serverHandler.Start(&wg)

	// Iniciar el watcher de archivos
	go h.watcher.FileWatcherStart(&wg)

	// Esperar a que todas las goroutines terminen
	wg.Wait()
}

func (h *handler) AddHandlers() {

	// LDFlags      func() []string // eg: []string{"-X 'main.version=v1.0.0'","-X 'main.buildDate=2023-01-01'"}

	//SERVER
	h.serverHandler = NewServerHandler(&ServerConfig{
		RootFolder:                  h.ch.config.WebFilesFolder,
		MainFileWithoutExtension:    "main.server",
		ArgumentsForCompilingServer: nil,
		ArgumentsToRunServer:        nil,
		PublicFolder:                h.ch.config.PublicFolder(),
		AppPort:                     h.ch.config.ServerPort,
		Print:                       h.tui.Print,
		ExitChan:                    h.exitChan,
	})

	//WASM
	h.wasmHandler = NewWasmCompiler(&WasmConfig{
		WebFilesFolder: func() (string, string) {
			return h.ch.config.WebFilesFolder, h.ch.config.PublicFolder()
		},
		Print: h.tui.Print,
	})

	//ASSETS
	h.assetsHandler = NewAssetsCompiler(&AssetsConfig{
		WebFilesFolder:         h.ch.config.OutPutStaticsDirectory,
		Print:                  h.tui.Print,
		WasmProjectTinyGoJsUse: h.wasmHandler.WasmProjectTinyGoJsUse,
	})

	// WATCHER
	h.watcher = NewWatchHandler(&WatchConfig{
		AppRootDir:                 h.ch.appRootDir,
		AssetsFileUpdateFileOnDisk: h.assetsHandler.NewFileEvent,
		GoFilesUpdateFileOnDisk:    h.serverHandler.NewFileEvent,
		WasmFilesUpdateFileOnDisk:  h.wasmHandler.NewFileEvent,
		BrowserReload:              h.browser.Reload,
		Print:                      h.tui.Print,
		ExitChan:                   h.exitChan,
		UnobservedFiles: func() []string {

			uf := []string{
				".git",
				".gitignore",
				".vscode",
				".exe",
			}

			uf = append(uf, h.assetsHandler.UnobservedFiles()...)
			uf = append(uf, h.wasmHandler.UnobservedFiles()...)
			uf = append(uf, h.serverHandler.UnobservedFiles()...)
			return uf
		},
	})

}
