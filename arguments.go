package godev

import (
	"fmt"
	"net/url"
	"os"
)

// path ej: http://localhost:8080/index.html"
// port ej: 8080
func (a *Args) CaptureArguments() {
	if len(os.Args) != 2 {
		fmt.Println("Se requiere: endpoint con puerto incluido para iniciar ej:")
		a.ShowErrorAndExit("godev http://localhost:8080/level_3.html")
	}

	parsedUrl, err := url.Parse(os.Args[1])
	if err != nil {
		a.ShowErrorAndExit("Error url parse " + err.Error())
	}
	a.Port = parsedUrl.Port()
	a.Path = os.Args[1]

	// fmt.Println("ARGUMENTOS:1 ", os.Args[0])
	// fmt.Println("PUERTO: ", a.Port, " PATH: ", a.Path, " ARGUMENTOS:2 ", os.Args[1])

}

//	for _, option := range os.Args {
//		if strings.Contains(option, "endpoint=") {
//			extractArgumentValue(option, &a.Endpoint)
//		} else if strings.Contains(option, "port=") {
//			extractArgumentValue(option, &a.Port)
//		}
//	}
// func extractArgumentValue(option string, field *string) {
// 	parts := strings.Split(option, "=")
// 	if len(parts) == 2 {
// 		*field = parts[1]
// 	} else {
// 		a.ShowErrorAndExit("Error: Delimitador '=' no encontrado en la cadena " + option)
// 	}
// }
