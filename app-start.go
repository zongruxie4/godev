package godev

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func (a *Args) StartProgram() {

	// Cambiar al directorio "cmd" si existe
	cmdDir := "cmd"
	if _, err := os.Stat(cmdDir); err == nil {
		err := os.Chdir(cmdDir)
		if err != nil {
			ShowErrorAndExit(fmt.Sprintf("Error al cambiar al directorio '%s': %s", cmdDir, err))
		}
	}

	a.Cmd = exec.Command("go", "run", "main.go")
	a.Cmd.Args = append(a.Cmd.Args, a.args...)

	stdoutPipe, err := a.Cmd.StdoutPipe()
	if err != nil {
		ShowErrorAndExit(fmt.Sprintf("Error al crear el pipe para la salida del programa: %s", err))
	}

	a.Scanner = bufio.NewScanner(stdoutPipe)

	err = a.Cmd.Start()
	if err != nil {
		ShowErrorAndExit(fmt.Sprintf("Error al iniciar el programa: %s", err))
	}

	time.Sleep(100 * time.Millisecond) // Esperar

}
