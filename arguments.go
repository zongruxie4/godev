package godev

import (
	"fmt"
	"os"
	"strings"
)

// path ej: http://localhost:8080/index.html"
// port ej: 8080
func (a *Args) CaptureArguments() {
	a.args = os.Args

	for i, opt := range a.args {

		switch {
		case strings.Contains(opt, "path:"):
			a.extractArgumentValue(opt, &a.browser_path)
			a.removeItem(i)

		case strings.Contains(opt, "port:"):
			a.app_port = true

		case strings.Contains(opt, "with:"):
			a.extractArgumentValue(opt, &a.with)
			a.removeItem(i)

		case strings.Contains(opt, "height:"):
			a.extractArgumentValue(opt, &a.height)
			a.removeItem(i)

		case strings.Contains(opt, "position:"):
			a.extractArgumentValue(opt, &a.position)
			a.removeItem(i)

		case opt == "help" || opt == "?" || opt == "ayuda":

			fmt.Println("default: port:8080 path:/")
			fmt.Println("*** ej valores admitidos***")
			fmt.Println("port:9090")
			fmt.Println("path:/login")
			fmt.Println("-----------------------")
			fmt.Println("--- Browser Options ---")
			fmt.Println("with:800")
			fmt.Println("height:600")
			fmt.Println("position:1930,0")
			fmt.Println("*-position es en caso de que tengas segundo monitor")
			a.ShowErrorAndExit("")
		}

	}

	if !a.app_port {
		a.args = append(a.args, "port:8080")
	}

	if a.browser_path == "" {
		a.browser_path = "/"
	}

}

func (a *Args) extractArgumentValue(option string, field *string) {
	parts := strings.Split(option, ":")
	if len(parts) == 2 {
		*field = parts[1]
	} else {
		a.ShowErrorAndExit("Error: Delimitador ':' no encontrado en la cadena " + option)
	}
}

func (a *Args) removeItem(index int) {
	a.args = append(a.args[:index], a.args[index+1:]...)
}
