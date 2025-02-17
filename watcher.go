package godev

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type WatchConfig struct {
	AppRootDir string // eg: "home/user/myNewApp"

	AssetsFileUpdateFileOnDisk func(filePath, extension, event string) error // when change assets files eg: css, js, html, png, jpg, svg, etc event: create, remove, write, rename
	GoFilesUpdateFileOnDisk    func(filePath, extension, event string) error // when change go files to backend or any destination
	WasmFilesUpdateFileOnDisk  func(filePath, extension, event string) error // when change go files to webAssembly destination

	BrowserReload func() error // when change frontend files reload browser

	Print           func(messages ...any) // eg: fmt.Println
	ExitChan        chan bool             // global channel to signal the exit
	UnobservedFiles func() []string       // files that are not observed by the watcher eg: ".git", ".gitignore", ".vscode",  "examples",
}

type WatchHandler struct {
	*WatchConfig
	watcher         *fsnotify.Watcher
	no_add_to_watch map[string]bool
}

func NewWatchHandler(c *WatchConfig) *WatchHandler {

	return &WatchHandler{
		WatchConfig: c,
	}
}

func (h *WatchHandler) FileWatcherStart(wg *sync.WaitGroup) {

	if h.watcher == nil {
		if watcher, err := fsnotify.NewWatcher(); err != nil {
			h.Print("Error New Watcher: " + err.Error())
			return
		} else {
			h.watcher = watcher
		}
	}

	// Start watching in the main routine
	go h.watchEvents()

	h.RegisterFiles()

	h.Print("Listening for File Changes ...")
	// Wait for exit signal after watching is active

	select {
	case <-h.ExitChan:
		h.watcher.Close()
		wg.Done()
		return
	}
}

func (h *WatchHandler) watchEvents() {
	lastActions := make(map[string]time.Time)

	reloadBrowserTimer := time.NewTimer(0)
	reloadBrowserTimer.Stop()

	restarTimer := time.NewTimer(0)
	restarTimer.Stop()

	var wait = 50 * time.Millisecond

	for {
		select {

		case event, ok := <-h.watcher.Events:
			if !ok {
				h.Print("Error h.watcher.Events")
				return
			}

			// h.Print("DEBUG Event:", event.Name, event.Op)
			// Aplicar debouncing para evitar múltiples eventos
			if lastTime, ok := lastActions[event.Name]; !ok || time.Since(lastTime) > 1*time.Second {

				// Restablece el temporizador de recarga de navegador
				reloadBrowserTimer.Stop()

				// Verificar si es un nuevo directorio para agregarlo al watcher
				if info, err := os.Stat(event.Name); err == nil && !h.Contain(event.Name) {

					// create, write, rename, remove
					eventType := strings.ToLower(event.Op.String())

					switch event.Op.String() {
					case "CREATE":
						h.watcher.Add(event.Name)
						h.Print("New directory created:", event.Name)
					case "REMOVE":
						h.watcher.Remove(event.Name)
						h.Print("Directory removed:", event.Name)
					}

					if !info.IsDir() {
						h.Print("Event type:", event.Op.String(), "File changed:", event.Name)

						extension := filepath.Ext(event.Name)
						// fmt.Println("extension:", extension, "File Event:", event)

						switch extension {

						case ".css", ".js", ".html":
							err = h.AssetsFileUpdateFileOnDisk(event.Name, extension, eventType)

						case ".go":
							var goFileName string
							goFileName, err = GetFileName(event.Name)
							if err == nil {

								isFrontend, isBackend := IsFileType(goFileName)

								if isFrontend { // compilar a wasm y recargar el navegador
									// h.Print("Go File IsFrontend")
									err = h.WasmFilesUpdateFileOnDisk(goFileName, event.Name, eventType)

								} else if isBackend { // compilar servidor y recargar el navegador
									// h.Print("Go File IsBackend")
									err = h.GoFilesUpdateFileOnDisk(goFileName, event.Name, eventType)

								} else { // ambos compilar servidor, compilar a wasm (según modulo) y recargar el navegador
									// h.Print("Go File Shared")
									err = h.WasmFilesUpdateFileOnDisk(goFileName, event.Name, eventType)
									if err == nil {
										err = h.GoFilesUpdateFileOnDisk(goFileName, event.Name, eventType)
									}

								}

							}

						default:
							err = errors.New("Watch Unknown file type: " + extension)
						}

						if err != nil {
							h.Print("Watch updating file:", err)
						} else {
							reloadBrowserTimer.Reset(wait)
						}

					}

					lastActions[event.Name] = time.Now()
				}

			}
		case err, ok := <-h.watcher.Errors:
			if !ok {
				h.Print("h.watcher.Errors:", err)
				return
			}

		case <-reloadBrowserTimer.C:
			// El temporizador de recarga ha expirado, ejecuta reload del navegador
			err := h.BrowserReload()
			if err != nil {
				h.Print("Watch:", err)
			}

		case <-h.ExitChan:
			h.watcher.Close()
			return
		}
	}
}

func (h *WatchHandler) RegisterFiles() {
	h.Print("RegisterFiles APP ROOT DIR: " + h.AppRootDir)

	reg := make(map[string]struct{})

	err := filepath.Walk(h.AppRootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			h.Print("accessing path:", path, err)
			return nil
		}

		if info.IsDir() && !h.Contain(path) {
			if _, exists := reg[path]; !exists {
				if err := h.watcher.Add(path); err != nil {
					h.Print("Error adding watch path:", path, err)
					return nil
				}
				reg[path] = struct{}{}
				h.Print("Watch path added:", path)
			}
		}
		return nil
	})

	if err != nil {
		h.Print("Walking directory:", err)
	}
}

func (h *WatchHandler) Contain(path string) bool {

	// ignore hidden files
	if strings.HasPrefix(filepath.Base(path), ".") {
		return true
	}

	if h.no_add_to_watch == nil {
		h.no_add_to_watch = map[string]bool{}

		// add files to ignore
		for _, file := range h.UnobservedFiles() {
			h.no_add_to_watch[file] = true
		}

	}

	for value := range h.no_add_to_watch {
		if strings.Contains(path, value) {
			return true
		}
	}

	return false
}
