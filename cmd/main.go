package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/cdvelop/godev"
	. "github.com/cdvelop/output"
	"github.com/chromedp/chromedp"
)

func main() {
	d := godev.Add()

	d.CompileAllProject()

	// Cree un canal para recibir señales de interrupción
	signal.Notify(d.Interrupt, os.Interrupt, syscall.SIGTERM)

	current_dir, err := os.Getwd()
	if err != nil {
		ShowErrorAndExit(err)
	}

	if filepath.Base(current_dir) == "godev" {
		ShowErrorAndExit("cambia al directorio de tu aplicación para ejecutar godev")
	}

	go d.StartProgram()

	var wg sync.WaitGroup
	wg.Add(2)

	go d.DevBrowserSTART(&wg)

	go d.DevFileWatcherSTART(&wg)

	<-d.Interrupt
	// Detenga el navegador y cierre la aplicación cuando se recibe una señal de interrupción
	if err := chromedp.Cancel(d.Context); err != nil {
		PrintError(fmt.Sprintf("al cerrar browser %v", err))
	}
	err = d.StopProgram()
	if err != nil {
		PrintError(fmt.Sprintf("al detener app: %v", err))
	}

	os.Exit(0)

}
