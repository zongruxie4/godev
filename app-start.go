package godev

import (
	"bufio"
	"fmt"
	"os/exec"
)

func (a *Args) StartProgram() {

	a.Cmd = exec.Command("go", "run", "main.go", "port="+a.Port)

	stdoutPipe, err := a.Cmd.StdoutPipe()
	if err != nil {
		a.ShowErrorAndExit(fmt.Sprintf("Error al crear el pipe para la salida del programa: %s", err))
	}

	a.Scanner = bufio.NewScanner(stdoutPipe)

	err = a.Cmd.Start()
	if err != nil {
		a.ShowErrorAndExit(fmt.Sprintf("Error al iniciar el programa: %s", err))
	}

	// fmt.Println("Programa iniciado exitosamente.")
}
