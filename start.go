package godev

import (
	"os"
	"sync"

	. "github.com/cdvelop/assetmin"
	. "github.com/cdvelop/devtui"
	"github.com/cdvelop/tinytranslator"
	. "github.com/cdvelop/tinytranslator"
	"github.com/cdvelop/tinywasm"
)

type handler struct {
	rootDir string      // Application root directory
	config  *AutoConfig // Main configuration source
	*Translator
	tui           *DevTUI
	serverHandler *ServerHandler
	assetsHandler *AssetMin
	wasmHandler   *tinywasm.TinyWasm
	watcher       *WatchHandler
	browser       *Browser
	exitChan      chan bool // Canal global para señalizar el cierre
}

func GodevStart(rootDir string, logger func(messages ...any)) {
	var err error
	h := &handler{
		rootDir:  rootDir,
		exitChan: make(chan bool),
	}

	// Validate we're not in system directories
	homeDir, _ := os.UserHomeDir()
	if rootDir == homeDir || rootDir == "/" {
		// Use the provided logger since Translator is not initialized yet
		logger("Cannot run godev in user root directory. Please run in a Go project directory")
		return
	}
	h.Translator, err = tinytranslator.NewTranslationEngine().WithCurrentDeviceLanguage()
	if err != nil {
		logger("Error initializing translator:", err)
		return
	}
	h.NewBrowser()

	h.tui = NewTUI(&TuiConfig{
		AppName:       "GODEV",
		TabIndexStart: 0,
		ExitChan:      h.exitChan,
		Color: &ColorStyle{
			Foreground: "#F4F4F4", // #F4F4F4
			Background: "#000000", // #000000
			Highlight:  "#FF6600", // #FF6600, FF6600  73ceddff
			Lowlight:   "#666666", // #666666
		},
		LogToFile: func(messageErr any) { logger(messageErr) },
	}) // Initialize AutoConfig FIRST - this will be our configuration source
	h.config = NewAutoConfig(logger) // Use the provided logger
	h.config.SetRootDir(h.rootDir)

	// Scan initial architecture - this must happen before AddSectionBUILD
	h.config.ScanDirectoryStructure()

	h.AddSectionBUILD()

	var wg sync.WaitGroup
	wg.Add(3)

	// Start the tui in a goroutine
	go h.tui.InitTUI(&wg)

	// Iniciar servidor
	go h.serverHandler.Start(&wg)

	// Iniciar el watcher de archivos
	go h.watcher.FileWatcherStart(&wg)

	// Esperar a que todas las goroutines terminen
	wg.Wait()
}
