package godev

import (
	"sync"

	. "github.com/cdvelop/devtui"
	"github.com/cdvelop/tinytranslator"
	. "github.com/cdvelop/tinytranslator"
)

type handler struct {
	*Translator
	ch            *ConfigHandler
	tui           *DevTUI
	serverHandler *ServerHandler
	assetsHandler *AssetsHandler
	wasmHandler   *WasmHandler
	watcher       *WatchHandler
	browser       *Browser
	exitChan      chan bool // Canal global para señalizar el cierre
}

func GodevStart() {
	var err error
	h := &handler{
		exitChan: make(chan bool),
	}

	h.Translator, err = tinytranslator.NewTranslationEngine().WithCurrentDeviceLanguage()
	if err != nil {
		h.LogToFile(err)
		return
	}

	h.NewConfig()
	h.NewBrowser()

	h.tui = NewTUI(&TuiConfig{
		AppName:       "GODEV",
		TabIndexStart: 0,
		ExitChan:      h.exitChan,
		Color: &ColorStyle{
			ForeGround: "#F4F4F4", // #F4F4F4
			Background: "#000000", // #000000
			Highlight:  "#FF6600", // #FF6600
			Lowlight:   "#666666", // #666666
		},
		LogToFile: h.LogToFile,
	})

	h.BuildTabHandlers()

	// h.tui = NewTUI(h.exitChan, h.serverHandler, h.assetsHandler, h.wasmHandler)

	var wg sync.WaitGroup
	wg.Add(3)

	// Start the tui in a goroutine
	go h.tui.InitTUI(&wg)

	// Mostrar errores de configuración como warning
	if len(h.ch.configErrors) != 0 {
		for _, err := range h.ch.configErrors {
			h.tui.Print(err)
		}
	}

	// Iniciar servidor
	go h.serverHandler.Start(&wg)

	// Iniciar el watcher de archivos
	go h.watcher.FileWatcherStart(&wg)

	// Esperar a que todas las goroutines terminen
	wg.Wait()
}
