package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cdvelop/godev"
	"github.com/chromedp/chromedp"
)

var a = godev.Args{
	ReloadBrowser: make(chan bool, 1),
	AppStop:       make(chan bool, 1),
	Interrupt:     make(chan os.Signal, 1),
}

func main() {
	a.CaptureArguments()

	dir, _ := os.Getwd()
	if filepath.Base(dir) == "godev" {
		a.ShowErrorAndExit("error cambia al directorio de tu aplicación para ejecutar godev")
	}

	a.StartProgram()

	a.StartDevSERVER()

	go a.DevBrowserSTART()

	// Cree un canal para recibir señales de interrupción
	signal.Notify(a.Interrupt, os.Interrupt, syscall.SIGTERM)

	<-a.Interrupt
	// Detenga el navegador y cierre la aplicación cuando se recibe una señal de interrupción
	if err := chromedp.Cancel(a.Context); err != nil {
		log.Println("error al cerrar browser", err)
	}
	a.StopProgram()
	os.Exit(0)

}
