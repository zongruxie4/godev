package godev

import "fmt"

func (a *Args) ProcessProgramOutput() {

	for a.Scanner.Scan() {
		line := a.Scanner.Text()

		switch line {

		case "restart_app":

			a.StopProgram()

			a.StartProgram()

			a.ReloadBrowser <- true

		case "reload_browser":
			a.ReloadBrowser <- true

		default:
			fmt.Println(line)
		}

		// Verificar si se ha enviado una señal de finalización
		select {
		case <-a.AppStop:
			fmt.Println("CERRANDO GO RUTINA ProcessProgramOutput")
			return // Salir de la rutina
		default:
			// Continuar el bucle
		}
	}
}
