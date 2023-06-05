package godev

import (
	"fmt"
	"log"
	"sync"

	"github.com/chromedp/chromedp"
)

// func (a *Args) ProcessProgramOutput() {

// 	for a.Scanner.Scan() {
// 		line := a.Scanner.Text()

// 		switch line {

// 		case "restart_app":

// 			a.StopProgram()

// 			a.StartProgram()

// 			// a.ReloadBrowser <- true
// 			a.reloadBrowser()

// 		case "reload_browser":
// 			// a.ReloadBrowser <- true
// 			a.reloadBrowser()

// 		default:
// 			fmt.Println(line)
// 		}

// 		// Verificar si se ha enviado una señal de finalización
// 		// for {
// 		// select {
// 		<-a.AppStop
// 		fmt.Println("CERRANDO GO RUTINA ProcessProgramOutput")
// 		return // Salir de la rutina
// 		// default:
// 		// Continuar el bucle
// 		// }
// 		// }
// 	}
// }

func (a *Args) ProcessProgramOutputOLD() {
	for {
		select {
		case <-a.AppStop:
			fmt.Println("CERRANDO GO RUTINA ProcessProgramOutput")
			return // Salir de la rutina

		default:
			if a.Scanner.Scan() {
				line := a.Scanner.Text()

				switch line {
				case "restart_app":
					a.StopProgram()
					a.StartProgram()
					a.reloadBrowser()

				case "reload_browser":
					a.reloadBrowser()

				default:
					fmt.Println(line)
				}
			}
		}
	}
}

func (a *Args) ProcessProgramOutput(wg *sync.WaitGroup) {
	defer wg.Done()

	for a.Scanner.Scan() {
		line := a.Scanner.Text()

		switch line {
		case "restart_app":
			a.StopProgram()
			a.StartProgram()
			a.reloadBrowser()

		case "reload_browser":
			a.reloadBrowser()

		default:
			fmt.Println(line)
		}
	}

}

func (a *Args) reloadBrowser() {
	// fmt.Println("Recargando Navegador")
	err := chromedp.Run(a.Context, chromedp.Reload())
	if err != nil {
		log.Println("Error al recargar Pagina ", err)
	}

}
