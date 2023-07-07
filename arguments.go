package godev

import (
	"fmt"
	"os"
	"strings"
)

// path: ej: /index.html, / ,/home
// port: ej: 9090
// domain ej: 192.0.0.5,app.com default localhost
func (a *Args) CaptureArguments() {
	a.args = os.Args

	var new_args []string

	for _, opt := range a.args {

		switch {
		case strings.Contains(opt, "path:"):
			extractArgumentValue(opt, &a.path)
			continue

		case strings.Contains(opt, "port:"):
			extractArgumentValue(opt, &a.port)

		case strings.Contains(opt, "domain:"):
			extractArgumentValue(opt, &a.domain)

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
			fmt.Println("protocol:https default http")
			fmt.Println("domain:/192.168.0.2 default localhost")
			fmt.Println("port:9090 default 8080")
			fmt.Println("path:/login,/home default /")
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

	if a.port == "" {
		a.port = "8080"
		a.args = append(a.args, "port:8080")
	}

	if a.path == "" {
		a.path = "/"
	}

	if a.domain == "" {
		a.domain = "localhost"
	}

	if a.protocol == "" {
		a.protocol = "http"
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
