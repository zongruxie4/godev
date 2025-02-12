package godev

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (h *handler) NewWatcher() {

	if watcher, err := fsnotify.NewWatcher(); err != nil {
		log.Fatal(errors.New("Error New Watcher: " + err.Error()))
	} else {
		h.watcher = watcher
	}
}

func (h *handler) FileWatcherStart(wg *sync.WaitGroup) {

	if h.watcher == nil {
		h.tui.PrintError("No file watcher found")
		return
	}

	// Start watching in the main routine
	go h.watchEvents()

	h.RegisterFiles()

	h.tui.PrintOK("Listening for File Changes ... ")
	// Wait for exit signal after watching is active

	select {
	case <-h.exitChan:
		wg.Done()
		return
	}
}

func (h *handler) watchEvents() {
	lastActions := make(map[string]time.Time)

	reloadBrowserTimer := time.NewTimer(0)
	reloadBrowserTimer.Stop()

	restarTimer := time.NewTimer(0)
	restarTimer.Stop()

	var wait = 50 * time.Millisecond

	for {
		select {

		case <-h.exitChan:
			h.watcher.Close()
			return

		case event, ok := <-h.watcher.Events:
			if !ok {
				h.tui.PrintError("Error h.watcher.Events")
				return
			}

			// Aplicar debouncing para evitar múltiples eventos
			if lastTime, ok := lastActions[event.Name]; !ok || time.Since(lastTime) > 1*time.Second {

				// Restablece el temporizador de recarga de navegador
				reloadBrowserTimer.Stop()

				// Verificar si es un nuevo directorio para agregarlo al watcher
				if info, err := os.Stat(event.Name); err == nil && !h.Contain(event.Name) {

					switch event.Op.String() {
					case "CREATE":
						h.watcher.Add(event.Name)
						h.tui.Print("New directory created:", event.Name)
					case "REMOVE":
						h.watcher.Remove(event.Name)
						h.tui.Print("Directory removed:", event.Name)
					}

					if !info.IsDir() {
						h.tui.Print("Event type:", event.Op.String(), "File changed:", event.Name)

						extension := filepath.Ext(event.Name)
						// fmt.Println("extension:", extension, "File Event:", event)

						switch extension {
						case ".html":
							// h.tui.PrintOK("HTML File")
							// err = w.HTML.UpdateFileOnDisk(event)

							// if err == nil {
							// 	resetWaitingTime = true
							// }
						case ".css", ".js":
							err = h.assetsCompiler.UpdateFileOnDisk(event.Name, extension)
							if err == nil {
								reloadBrowserTimer.Reset(wait)
							}

						case ".go":
							var goFileName string
							goFileName, err = GetFileName(event.Name)
							if err == nil {
								h.tui.Print("Go File:", goFileName)
							}

							h.tui.PrintOK("Go File")

						default:
							h.tui.PrintWarning("Watch Unknown file type:", extension)

						}

						if err != nil {
							h.tui.PrintError("Watch updating file:", err)
						}

					}

					lastActions[event.Name] = time.Now()
				}

			}
		case err, ok := <-h.watcher.Errors:
			if !ok {
				h.tui.PrintError("h.watcher.Errors:", err)
				return
			}

		case <-reloadBrowserTimer.C:
			// El temporizador de recarga ha expirado, ejecuta reload del navegador
			err := h.browser.Reload()
			if err != nil {
				h.tui.PrintError("Watch:", err)
			}
		}
	}
}

func (h *handler) RegisterFiles() {
	h.tui.PrintOK("RegisterFiles APP ROOT DIR: " + h.ch.appRootDir)

	reg := make(map[string]struct{})

	err := filepath.Walk(h.ch.appRootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			h.tui.PrintError("Error accessing path:", path, err)
			return nil
		}

		if info.IsDir() && !h.Contain(path) {
			if _, exists := reg[path]; !exists {
				if err := h.watcher.Add(path); err != nil {
					h.tui.PrintError("Error adding watch path:", path, err)
					return nil
				}
				reg[path] = struct{}{}
				h.tui.Print("Watch path added:", path)
			}
		}
		return nil
	})

	if err != nil {
		h.tui.PrintError("Error walking directory:", err)
	}
}

var no_add_to_watch map[string]bool

func (h *handler) Contain(path string) bool {

	// Ignorar archivos temporales o hidden
	if strings.HasPrefix(filepath.Base(path), ".") {
		return true
	}

	if no_add_to_watch == nil {
		no_add_to_watch := map[string]bool{
			".git":    true,
			".vscode": true,
			".exe":    true,
		}

		// ignorar archivos generados por el compilador de assets como script.js, style.css
		for _, file := range h.assetsCompiler.UnchangeableOutputFileNames() {
			no_add_to_watch[file] = true
		}

		// ignorar archivos generados por wasm compiler
		for _, file := range h.wasmCompiler.UnchangeableOutputFileNames() {
			no_add_to_watch[file] = true
		}

	}

	for value := range no_add_to_watch {
		if strings.Contains(path, value) {
			return true
		}
	}

	return false
}
