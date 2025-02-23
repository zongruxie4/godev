package godev

import (
	"path"
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
	h.NewBrowser()
	h.AddHandlers()

	h.tui = NewTUI(1, h.exitChan)
	// h.tui = NewTUI(h.exitChan, h.serverHandler, h.assetsHandler, h.wasmHandler)

	var wg sync.WaitGroup
	wg.Add(3)

	// Start the tui in a goroutine
	go h.tui.StartTUI(&wg)

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
		Writer:                      h,
		ExitChan:                    h.exitChan,
	})

	//WASM
	h.wasmHandler = NewWasmCompiler(&WasmConfig{
		WebFilesFolder: func() (string, string) {
			return h.ch.config.WebFilesFolder, h.ch.config.PublicFolder()
		},
		Writer: h,
	})

	//ASSETS
	h.assetsHandler = NewAssetsCompiler(&AssetsConfig{
		ThemeFolder: func() string {
			return path.Join(h.ch.config.WebFilesFolder, "theme")
		},
		WebFilesFolder: h.ch.config.OutPutStaticsDirectory,
		Print:          h.tui.Print,
		JavascriptForInitializing: func() (string, error) {
			return h.wasmHandler.JavascriptForInitializing()
		},
	})

	// WATCHER
	h.watcher = NewWatchHandler(&WatchConfig{
		AppRootDir:      h.ch.appRootDir,
		FileEventAssets: h.assetsHandler,
		FileEventGO:     h.serverHandler,
		FileEventWASM:   h.wasmHandler,
		FileTypeGO: GoFileType{
			FrontendPrefix: []string{"f."},
			FrontendFiles: []string{
				h.wasmHandler.mainOutputFile,
			},
			BackendPrefix: []string{"b."},
			BackendFiles: []string{
				h.serverHandler.mainFileExternalServer,
			},
		},
		BrowserReload: h.browser.Reload,

		Writer:   h,
		ExitChan: h.exitChan,
		UnobservedFiles: func() []string {

			uf := []string{
				".git",
				".gitignore",
				".vscode",
				".exe",
				".log",
				"_test.go",
			}

			uf = append(uf, h.assetsHandler.UnobservedFiles()...)
			uf = append(uf, h.wasmHandler.UnobservedFiles()...)
			uf = append(uf, h.serverHandler.UnobservedFiles()...)
			return uf
		},
	})

}
