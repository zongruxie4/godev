package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/cdvelop/godev"
	"github.com/chromedp/chromedp"
)

var a = godev.Args{
	Interrupt: make(chan os.Signal, 1),
}

func main() {
	a.CaptureArguments()

	// Cree un canal para recibir señales de interrupción
	signal.Notify(a.Interrupt, os.Interrupt, syscall.SIGTERM)

	current_dir, err := os.Getwd()
	if err != nil {
		godev.ShowErrorAndExit(err.Error())
	}

	if filepath.Base(current_dir) == "godev" {
		godev.ShowErrorAndExit("cambia al directorio de tu aplicación para ejecutar godev")
	}

	a.RegisterFoldersPackages(current_dir)

	a.StartProgram()

	var wg sync.WaitGroup
	wg.Add(1)

	go a.ProcessProgramOutput(&wg)

	// go a.StartDevSERVER(&wg)

	// go a.DevBrowserSTART(&wg)

	<-a.Interrupt
	// Detenga el navegador y cierre la aplicación cuando se recibe una señal de interrupción
	if err := chromedp.Cancel(a.Context); err != nil {
		log.Println("error al cerrar browser", err)
	}
	a.StopProgram()
	os.Exit(0)

}
