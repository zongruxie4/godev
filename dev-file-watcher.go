package godev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var folders_watch = []string{"modules", "ui\\theme"}

// La función principal es donde se crea el observador para monitorear los cambios en los archivos y directorios.
// En esta función, también configuraremos los filtros para los tipos de archivo que queremos observar.
func (u ui) DevFileWatcherSTART(wg *sync.WaitGroup) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer watcher.Close()

	go u.watchEvents(watcher, wg)

	for _, folder := range folders_watch {

		filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				// fmt.Println(path)
				watcher.Add(path)
			}
			return nil
		})
	}

	fmt.Println("Escuchando Eventos UI ...")
	select {}
}

func (u ui) watchEvents(watcher *fsnotify.Watcher, wg *sync.WaitGroup) {
	defer wg.Done()
	last_actions := make(map[string]time.Time)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if last_time, ok := last_actions[event.Name]; !ok || time.Since(last_time) >= time.Second {
				// Registrar la última acción y procesar el evento.
				last_actions[event.Name] = time.Now()

				if isDir(event.Name) {
					// fmt.Println("Folder Event:", event.Name)
				} else {
					// fmt.Println("File Event:", event.Name)

					extension := filepath.Ext(event.Name)

					switch extension {
					case ".css":
						fmt.Println("Compilando CSS...", event.Name)
						u.BuildCSS()
						// RELOADED HERE
						u.reload <- true
					case ".js":
						fmt.Println("Compilando JS...", event.Name)
						u.BuildJS()
						// RELOADED HERE
						u.reload <- true
					case ".html":
						fmt.Println("Compilando HTML...", event.Name)
						u.BuildHTML()
						// RELOADED HERE

						u.reload <- true
					case ".go":

						if strings.Contains(event.Name, "wasm") {
							fmt.Println("Compilando WASM...", event.Name)
							u.BuildWASM()
							// RELOADED HERE

							u.reload <- true
						}

					}
				}

			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("Error:", err)

		case <-u.reload:
			fmt.Println("Leyendo señal de recarga del canal")

		}
	}

}
