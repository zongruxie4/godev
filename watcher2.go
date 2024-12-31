package godev

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func (h *handler) FileWatcherStart(wg *sync.WaitGroup) {
	defer wg.Done()

	if h.watcher == nil {
		h.terminal.MsgError("No file watcher found")
		return
	}

	h.RegisterFiles()

	done := make(chan bool)
	go h.watchEvents(done)
	defer h.watcher.Close()

	h.terminal.MsgOk("Listening for File Changes ... ")

	// Esperar señal de cierre
	<-exitChan
}

func (h *handler) watchEvents(done chan bool) {
	lastActions := make(map[string]time.Time)

	for {
		select {
		case <-exitChan:
			return

		case event, ok := <-h.watcher.Events:
			if !ok {
				done <- true
				return
			}

			// Ignorar archivos temporales o hidden
			if strings.HasPrefix(filepath.Base(event.Name), ".") {
				continue
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
					}

					lastActions[event.Name] = time.Now()
				}

			}
		case err, ok := <-h.watcher.Errors:
			if !ok {
				done <- true
				return
			}
			h.terminal.MsgError("Watcher error:", err)
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
	var no_add_to_watch = []string{".devcontainer", ".git", ".vscode", config.OutputDir}

	for _, value := range no_add_to_watch {
		if strings.Contains(path, value) {
			return true
		}
	}

	return false
}
