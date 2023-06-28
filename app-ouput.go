package godev

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/chromedp/chromedp"
)

func (a *Args) ProcessProgramOutput(wg *sync.WaitGroup) {
	defer wg.Done()

	for a.Scanner.Scan() {
		line := a.Scanner.Text()

		switch {
		case strings.Contains(line, "restart_app"):
			a.StopProgram()
			a.StartProgram()
			a.reloadBrowser()

		case strings.Contains(line, "reload_browser"):
			a.reloadBrowser()

		// case strings.Contains(line, "module:"):

		// var module string
		// extractArgumentValue(line, &module)

		// for _, m := range modules {
		// fmt.Println("MODULO RECIBIDO:", module)
		// }

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
