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

	Print           func(messages ...any) // For logging output
	ExitChan        chan bool             // global channel to signal the exit
	UnobservedFiles func() []string       // files that are not observed by the watcher eg: ".git", ".gitignore", ".vscode",  "examples",
}

type WatchHandler struct {
	*WatchConfig
	watcher         *fsnotify.Watcher
	no_add_to_watch map[string]bool
	// logMu           sync.Mutex // No longer needed with Print func
}

func NewWatchHandler(c *WatchConfig) *WatchHandler {

	return &WatchHandler{
		WatchConfig: c,
	}
}

func (h *WatchHandler) FileWatcherStart(wg *sync.WaitGroup) {

	if h.watcher == nil {
		if watcher, err := fsnotify.NewWatcher(); err != nil {
			h.Print("Error New Watcher: ", err)
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
	// Restored but with shorter debounce time to avoid missing important events
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
			// Restore debouncer with shorter timeout - 100ms is enough for file operations to complete,
			// but short enough to not miss important events like CREATE followed by WRITE
			if lastTime, ok := lastActions[event.Name]; !ok || time.Since(lastTime) > 100*time.Millisecond {

				// Restablece el temporizador de recarga de navegador
				reloadBrowserTimer.Stop()

				// Verificar si es un nuevo directorio para agregarlo al watcher
				if info, err := os.Stat(event.Name); err == nil && !h.Contain(event.Name) && !info.IsDir() {

					// create, write, rename, remove
					eventType := strings.ToLower(event.Op.String())
					// h.Print("Event type:", event.Op.String(), "File changed:", event.Name)

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
								// h.Print("Go File IsFrontend")
								err = h.FileEventWASM.NewFileEvent(fileName, extension, event.Name, eventType)

							} else if isBackend { // compilar servidor y recargar el navegador
								// h.Print("Go File IsBackend")
								err = h.FileEventGO.NewFileEvent(fileName, extension, event.Name, eventType)

							} else { // ambos compilar servidor, compilar a wasm (según modulo) y recargar el navegador
								// h.Print("Go File Shared")
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
						h.Print("Watch updating file:", err)
					} else {
						reloadBrowserTimer.Reset(wait)
					}

					// Update the last action time for debouncing
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
					h.Print("Watch RegisterFiles Add watch path:", path, err)
					return nil
				}
				reg[path] = struct{}{}
				h.Print("Watch path added:", path)

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
					h.Print("Watch RegisterFiles:", extension, err)
				}

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

		// add files to ignore only if UnobservedFiles is configured
		if h.UnobservedFiles != nil {
			for _, file := range h.UnobservedFiles() {
				h.no_add_to_watch[file] = true
			}
		}

	}

	// Check for exact match against the full paths in the ignore list
	if _, exists := h.no_add_to_watch[path]; exists {
		return true
	}

	// Additionally, ignore directories within ignored paths (e.g., subfolders of .git)
	for ignoredPath := range h.no_add_to_watch {
		// Check if the current path starts with an ignored path + separator
		// This prevents watching subdirectories of ignored directories (like .git/hooks)
		if strings.HasPrefix(path, ignoredPath+string(filepath.Separator)) {
			return true
		}
	}

	return false
}
