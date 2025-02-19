package godev

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// when change go files to webAssembly destination
type FileTypeGO interface {
	GoFileIsType(fileName string) (isFrontend, isBackend bool)
}

// event: create, remove, write, rename
type FileEvent interface {
	NewFileEvent(fileName, extension, filePath, event string) error
}

type WatchConfig struct {
	AppRootDir string // eg: "home/user/myNewApp"

	FileEventAssets FileEvent // when change assets files eg: css, js, html, png, jpg, svg, etc event: create, remove, write, rename
	FileEventGO     FileEvent // when change go files to backend or any destination
	FileEventWASM   FileEvent // when change go files to webAssembly destination

	FileTypeGO // when change go files to webAssembly destination

	BrowserReload func() error // when change frontend files reload browser

	Writer          io.Writer       // For logging output
	ExitChan        chan bool       // global channel to signal the exit
	UnobservedFiles func() []string // files that are not observed by the watcher eg: ".git", ".gitignore", ".vscode",  "examples",
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
			fmt.Fprintln(h.Writer, "Error New Watcher: ", err)
			return
		} else {
			h.watcher = watcher
		}
	}

	// Start watching in the main routine
	go h.watchEvents()

	h.RegisterFiles()

	fmt.Fprintln(h.Writer, "Listening for File Changes ...")
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
				fmt.Fprintln(h.Writer, "Error h.watcher.Events")
				return
			}

			// fmt.Fprintln(h.Writer, "DEBUG Event:", event.Name, event.Op)
			// Aplicar debouncing para evitar múltiples eventos
			if lastTime, ok := lastActions[event.Name]; !ok || time.Since(lastTime) > 1*time.Second {

				// Restablece el temporizador de recarga de navegador
				reloadBrowserTimer.Stop()

				// Verificar si es un nuevo directorio para agregarlo al watcher
				if info, err := os.Stat(event.Name); err == nil && !h.Contain(event.Name) && !info.IsDir() {

					// create, write, rename, remove
					eventType := strings.ToLower(event.Op.String())
					// fmt.Fprintln(h.Writer, "Event type:", event.Op.String(), "File changed:", event.Name)

					fileName, err := GetFileName(event.Name)
					if err == nil {

						extension := filepath.Ext(event.Name)
						// fmt.Println("extension:", extension, "File Event:", event)

						switch extension {

						case ".css", ".js", ".html":
							err = h.FileEventAssets.NewFileEvent(fileName, extension, event.Name, eventType)

						case ".go":

							isFrontend, isBackend := h.GoFileIsType(fileName)

							if isFrontend { // compilar a wasm y recargar el navegador
								// fmt.Fprintln(h.Writer, "Go File IsFrontend")
								err = h.FileEventWASM.NewFileEvent(fileName, extension, event.Name, eventType)

							} else if isBackend { // compilar servidor y recargar el navegador
								// fmt.Fprintln(h.Writer, "Go File IsBackend")
								err = h.FileEventGO.NewFileEvent(fileName, extension, event.Name, eventType)

							} else { // ambos compilar servidor, compilar a wasm (según modulo) y recargar el navegador
								// fmt.Fprintln(h.Writer, "Go File Shared")
								err = h.FileEventWASM.NewFileEvent(fileName, extension, event.Name, eventType)
								if err == nil {
									err = h.FileEventGO.NewFileEvent(fileName, extension, event.Name, eventType)
								}
							}

						default:
							err = errors.New("Watch Unknown file type: " + extension)
						}
					}

					if err != nil {
						fmt.Fprintln(h.Writer, "Watch updating file:", err)
					} else {
						reloadBrowserTimer.Reset(wait)
					}

					lastActions[event.Name] = time.Now()
				}

			}
		case err, ok := <-h.watcher.Errors:
			if !ok {
				fmt.Fprintln(h.Writer, "h.watcher.Errors:", err)
				return
			}

		case <-reloadBrowserTimer.C:
			// El temporizador de recarga ha expirado, ejecuta reload del navegador
			err := h.BrowserReload()
			if err != nil {
				fmt.Fprintln(h.Writer, "Watch:", err)
			}

		case <-h.ExitChan:
			h.watcher.Close()
			return
		}
	}
}

func (h *WatchHandler) RegisterFiles() {
	fmt.Fprintln(h.Writer, "RegisterFiles APP ROOT DIR: "+h.AppRootDir)

	reg := make(map[string]struct{})

	err := filepath.Walk(h.AppRootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintln(h.Writer, "accessing path:", path, err)
			return nil
		}

		if info.IsDir() && !h.Contain(path) {
			if _, exists := reg[path]; !exists {
				if err := h.watcher.Add(path); err != nil {
					fmt.Fprintln(h.Writer, "Watch RegisterFiles Add watch path:", path, err)
					return nil
				}
				reg[path] = struct{}{}
				fmt.Fprintln(h.Writer, "Watch path added:", path)

				// MEMORY REGISTER FILES IN HANDLERS
				fileName, err := GetFileName(path)
				extension := filepath.Ext(path)
				if err == nil {

					switch extension {
					case ".html", ".css", ".js", ".svg":
						err = h.FileEventAssets.NewFileEvent(fileName, extension, path, "create")
					}
				}

				if err != nil {
					fmt.Fprintln(h.Writer, "Watch RegisterFiles:", extension, err)
				}

			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintln(h.Writer, "Walking directory:", err)
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
