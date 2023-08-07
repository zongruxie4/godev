package godev

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/cdvelop/gotools"
)

func (c *Dev) StartProgram() {

	// Cambiar al directorio "cmd" si existe
	cmdDir := "cmd"
	if _, err := os.Stat(cmdDir); err == nil {
		err := os.Chdir(cmdDir)
		if err != nil {
			gotools.ShowErrorAndExit(fmt.Sprintf("Error al cambiar al directorio '%s': %s", cmdDir, err))
		}
	}

	c.Cmd = exec.Command("go", "run", "main.go")
	c.Cmd.Args = append(c.Cmd.Args, c.args...)

	stdoutPipe, err := c.Cmd.StdoutPipe()
	if err != nil {
		gotools.ShowErrorAndExit(fmt.Sprintf("Error al crear el pipe para la salida del programa: %s", err))
	}

	c.Scanner = bufio.NewScanner(stdoutPipe)

	err = c.Cmd.Start()
	if err != nil {
		gotools.ShowErrorAndExit(fmt.Sprintf("Error al iniciar el programa: %s", err))
	}

	time.Sleep(100 * time.Millisecond) // Esperar

}
