package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cdvelop/godev"
	"github.com/chromedp/chromedp"
)

var a = godev.Args{
	RunBrowser: make(chan bool, 1),
	Reload:     make(chan bool, 1),
	Interrupt:  make(chan os.Signal, 1),
}

func main() {
	a.CaptureArguments()

	dir, _ := os.Getwd()
	if filepath.Base(dir) == "godev" {
		a.ShowErrorAndExit("error cambia al directorio de tu aplicación para ejecutar godev")
	}

	go a.ProcessProgramOutput()

	// Set up TCP server to listen for restart messages
	ln, err := net.Listen("tcp", ":1234") // Change ":1234" to the desired port number
	if err != nil {
		fmt.Printf("Failed to start TCP server: %s\n", err)
		return
	}
	defer ln.Close()

	// var wg sync.WaitGroup
	// wg.Add(2)

	a.TcpHandler(ln)

	a.StartProgram()

	// Cree un canal para recibir señales de interrupción
	signal.Notify(a.Interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-a.RunBrowser:
			go a.DevBrowserSTART()

		case <-a.Interrupt:
			// Detenga el navegador y cierre la aplicación cuando se recibe una señal de interrupción
			if err := chromedp.Cancel(a.Context); err != nil {
				log.Println("error al cerrar browser", err)
			}
			a.StopProgram()
			os.Exit(0)
		}

	}
}
