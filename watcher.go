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
		h.tui.MsgError("No file watcher found")
		return
	}

	// Start watching in the main routine
	go h.watchEvents()

	h.RegisterFiles()

	h.tui.MsgOk("Listening for File Changes ... ")
	// Wait for exit signal after watching is active

	select {
	case <-exitChan:
		wg.Done()
		return
	}
}

func (h *handler) watchEvents() {
	lastActions := make(map[string]time.Time)

	for {
		select {

		case <-exitChan:
			h.watcher.Close()
			return

		case event, ok := <-h.watcher.Events:
			if !ok {
				h.tui.MsgError("Error h.watcher.Events")
				return
			}

			// Aplicar debouncing para evitar múltiples eventos
			if lastTime, ok := lastActions[event.Name]; !ok || time.Since(lastTime) > 1*time.Second {

				// Verificar si es un nuevo directorio para agregarlo al watcher
				if info, err := os.Stat(event.Name); err == nil && !h.Contain(event.Name) {

					switch event.Op.String() {
					case "CREATE":
						h.watcher.Add(event.Name)
						h.tui.Msg("New directory created:", event.Name)
					case "REMOVE":
						h.watcher.Remove(event.Name)
						h.tui.Msg("Directory removed:", event.Name)
					}

					if !info.IsDir() {
						h.tui.Msg("Event type:", event.Op.String(), "File changed:", event.Name)

						extension := filepath.Ext(event.Name)
						// fmt.Println("extension:", extension, "File Event:", event)

						switch extension {
						case ".html":
							h.tui.MsgOk("HTML File")
							// err = w.HTML.UpdateFileOnDisk(event)

							// if err == nil {
							// 	resetWaitingTime = true
							// }
						case ".css":
							h.tui.MsgOk("CSS File")
							// err = w.CSS.UpdateFileOnDisk(event)
							// if err == nil {
							// 	resetWaitingTime = true
							// }
						case ".js":
							h.tui.MsgOk("JS File")
							// err = w.JS.UpdateFileOnDisk(event)
							// if err == nil {
							// 	resetWaitingTime = true
							// }

						case ".go":

							h.tui.MsgOk("Go File")

						}

					}

					lastActions[event.Name] = time.Now()
				}

			}
		case err, ok := <-h.watcher.Errors:
			if !ok {
				h.tui.MsgError("h.watcher.Errors:", err)
				return
			}
		}
	}
}

func (h *handler) RegisterFiles() {
	h.tui.MsgOk("RegisterFiles APP ROOT DIR: " + h.ch.appRootDir)

	reg := make(map[string]struct{})

	err := filepath.Walk(h.ch.appRootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			h.tui.MsgError("Error accessing path:", path, err)
			return nil
		}

		if info.IsDir() && !h.Contain(path) {
			if _, exists := reg[path]; !exists {
				if err := h.watcher.Add(path); err != nil {
					h.tui.MsgError("Error adding watch path:", path, err)
					return nil
				}
				reg[path] = struct{}{}
				h.tui.Msg("Watch path added:", path)
			}
		}
		return nil
	})

	if err != nil {
		h.tui.MsgError("Error walking directory:", err)
	}
}

func (h *handler) Contain(path string) bool {

	// Ignorar archivos temporales o hidden
	if strings.HasPrefix(filepath.Base(path), ".") {
		return true
	}

	var no_add_to_watch = []string{h.ch.config.OutputDir}

	for _, value := range no_add_to_watch {
		if strings.Contains(path, value) {
			return true
		}
	}

	return false
}
