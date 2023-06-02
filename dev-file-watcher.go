package godev

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var folders_watch = []string{"modules", "ui\\theme"}

// La función principal es donde se crea el observador para monitorear los cambios en los archivos y directorios.
// En esta función, también configuraremos los filtros para los tipos de archivo que queremos observar.
func (u ui) DevFileWatcherSTART() {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer watcher.Close()

	go u.watchEvents(watcher)

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

func (u ui) watchEvents(watcher *fsnotify.Watcher) {
	// defer wg.Done()
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
						sendTcpMessageReloadRestart(false)
					case ".js":
						fmt.Println("Compilando JS...", event.Name)
						u.BuildJS()
						// RELOADED HERE
						sendTcpMessageReloadRestart(false)
					case ".html":
						fmt.Println("Compilando HTML...", event.Name)
						u.BuildHTML()
						// RELOADED HERE

						sendTcpMessageReloadRestart(false)
					case ".go":

						if strings.Contains(event.Name, "wasm") {
							fmt.Println("Compilando WASM...", event.Name)
							u.BuildWASM()
							// RELOADED HERE

							sendTcpMessageReloadRestart(false)
						} else {
							sendTcpMessageReloadRestart(true)

						}

					}
				}

			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("Error:", err)
		}
	}

}

func sendTcpMessageReloadRestart(restart bool) {
	conn, err := net.Dial("tcp", "localhost:1234") // Dirección y puerto en los que el programa B está escuchando
	if err != nil {
		log.Println("Error Dial Tcp ", err)
	}
	defer conn.Close()

	var message string

	if restart {
		message = "restart"
	} else {
		message = "reload"
	}

	_, err = conn.Write([]byte(message))
	if err != nil {
		log.Println("Error al escribir mensaje tcp ", message, err)
	}

}
