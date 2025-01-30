package godev

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func (h *handler) FileWatcherStart(wg *sync.WaitGroup) {

	if h.watcher == nil {
		h.terminal.MsgError("No file watcher found")
		return
	}

	// Start watching in the main routine
	go h.watchEvents()

	h.RegisterFiles()

	h.terminal.MsgOk("Listening for File Changes ... ")
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
				h.terminal.MsgError("Error h.watcher.Events")
				return
			}

			// Aplicar debouncing para evitar múltiples eventos
			if lastTime, ok := lastActions[event.Name]; !ok || time.Since(lastTime) > 1*time.Second {

				// Verificar si es un nuevo directorio para agregarlo al watcher
				if info, err := os.Stat(event.Name); err == nil && !h.Contain(event.Name) {

					switch event.Op.String() {
					case "CREATE":
						h.watcher.Add(event.Name)
						h.terminal.Msg("New directory created:", event.Name)
					case "REMOVE":
						h.watcher.Remove(event.Name)
						h.terminal.Msg("Directory removed:", event.Name)
					}

					if !info.IsDir() {
						h.terminal.Msg("Event type:", event.Op.String(), "File changed:", event.Name)

						extension := filepath.Ext(event.Name)
						// fmt.Println("extension:", extension, "File Event:", event)

						switch extension {
						case ".html":
							h.terminal.MsgOk("HTML File")
							// err = w.HTML.UpdateFileOnDisk(event)

							// if err == nil {
							// 	resetWaitingTime = true
							// }
						case ".css":
							h.terminal.MsgOk("CSS File")
							// err = w.CSS.UpdateFileOnDisk(event)
							// if err == nil {
							// 	resetWaitingTime = true
							// }
						case ".js":
							h.terminal.MsgOk("JS File")
							// err = w.JS.UpdateFileOnDisk(event)
							// if err == nil {
							// 	resetWaitingTime = true
							// }

						case ".go":

							h.terminal.MsgOk("Go File")

						}

					}

					lastActions[event.Name] = time.Now()
				}

			}
		case err, ok := <-h.watcher.Errors:
			if !ok {
				h.terminal.MsgError("h.watcher.Errors:", err)
				return
			}
		}
	}
}

func (h *handler) RegisterFiles() {
	h.terminal.MsgOk("RegisterFiles APP ROOT DIR: " + APP_ROOT_DIR)

	reg := make(map[string]struct{})

	err := filepath.Walk(APP_ROOT_DIR, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			h.terminal.MsgError("Error accessing path:", path, err)
			return nil
		}

		if info.IsDir() && !h.Contain(path) {
			if _, exists := reg[path]; !exists {
				if err := h.watcher.Add(path); err != nil {
					h.terminal.MsgError("Error adding watch path:", path, err)
					return nil
				}
				reg[path] = struct{}{}
				h.terminal.Msg("Watch path added:", path)
			}
		}
		return nil
	})

	if err != nil {
		h.terminal.MsgError("Error walking directory:", err)
	}
}

func (h handler) Contain(path string) bool {

	// Ignorar archivos temporales o hidden
	if strings.HasPrefix(filepath.Base(path), ".") {
		return true
	}

	var no_add_to_watch = []string{config.OutputDir}

	for _, value := range no_add_to_watch {
		if strings.Contains(path, value) {
			return true
		}
	}

	return false
}
