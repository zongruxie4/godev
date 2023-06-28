package godev

import (
	"fmt"
	"os"
	"strings"
)

// path ej: /index.html, / ,/home
// port ej: 8080
func (a *Args) CaptureArguments() {
	a.args = os.Args

	var new_args []string

	for _, opt := range a.args {

		switch {
		case strings.Contains(opt, "path:"):
			extractArgumentValue(opt, &a.browser_path)
			continue

		case strings.Contains(opt, "port:"):
			a.app_port = true

		case strings.Contains(opt, "with:"):
			extractArgumentValue(opt, &a.with)
			continue

		case strings.Contains(opt, "height:"):
			extractArgumentValue(opt, &a.height)
			continue

		case strings.Contains(opt, "position:"):
			extractArgumentValue(opt, &a.position)
			continue

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
			ShowErrorAndExit("")
		}

		new_args = append(new_args, opt)
	}

	if !a.app_port {
		a.args = append(a.args, "port:8080")
	}

	if a.browser_path == "" {
		a.browser_path = "/"
	}

	a.args = new_args

}

func extractArgumentValue(option string, field *string) {
	parts := strings.Split(option, ":")
	if len(parts) >= 2 {
		*field = parts[1]
	} else {
		ShowErrorAndExit("Delimitador ':' no encontrado en la cadena " + option)
	}
}
