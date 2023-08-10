package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/cdvelop/godev"
	"github.com/cdvelop/gotools"
	"github.com/chromedp/chromedp"
)

func main() {
	c := godev.Add()

	c.CompileAllProject()

	// Cree un canal para recibir señales de interrupción
	signal.Notify(c.Interrupt, os.Interrupt, syscall.SIGTERM)

	current_dir, err := os.Getwd()
	if err != nil {
		gotools.ShowErrorAndExit(err.Error())
	}

	if filepath.Base(current_dir) == "godev" {
		gotools.ShowErrorAndExit("cambia al directorio de tu aplicación para ejecutar godev")
	}

	go c.StartProgram()

	var wg sync.WaitGroup
	wg.Add(2)

	go c.DevBrowserSTART(&wg)

	go c.DevFileWatcherSTART(&wg)

	<-c.Interrupt
	// Detenga el navegador y cierre la aplicación cuando se recibe una señal de interrupción
	if err := chromedp.Cancel(c.Context); err != nil {
		log.Println("error al cerrar browser", err)
	}
	c.StopProgram()
	os.Exit(0)

}
