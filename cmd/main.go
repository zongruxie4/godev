package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/cdvelop/godev"
	"github.com/cdvelop/gomod"
	. "github.com/cdvelop/output"
	"github.com/chromedp/chromedp"
)

func main() {

	current_dir, err := os.Getwd()
	if err != nil {
		ShowErrorAndExit(err)
	}

	if filepath.Base(current_dir) == "godev" {
		ShowErrorAndExit("cambia al directorio de tu aplicación para ejecutar godev")
	}

	if gomod.CommandFound() {
		return
	}

	d := godev.Add()

	d.CompileAllProject()

	// Cree un canal para recibir señales de interrupción
	signal.Notify(d.Interrupt, os.Interrupt, syscall.SIGTERM)

	go d.StartProgram()

	var app_started bool

	for {

		select {

		case message := <-d.ProgramStartedMessages:

			if strings.Contains(strings.ToLower(message), "err") {
				PrintError(message)
			} else {
				PrintOK(message)

				// go d.DevBrowserSTART(&wg)

				// go d.DevFileWatcherSTART(&wg)

				if !app_started {

					var wg sync.WaitGroup
					wg.Add(2)

					go d.DevBrowserSTART(&wg)

					go d.DevFileWatcherSTART(&wg)

					app_started = true
				}
			}

		case <-d.Interrupt:
			// Detenga el navegador y cierre la aplicación cuando se recibe una señal de interrupción
			if er := chromedp.Cancel(d.Context); er != nil {
				PrintError("al cerrar browser " + er.Error())
			}
			err := d.StopProgram()
			if err != "" {
				PrintError("al detener app: " + err)
			}

			os.Exit(0)
		}

	}

}
