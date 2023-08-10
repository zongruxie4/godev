package godev

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/cdvelop/gotools"
)

func (c *Dev) StartProgramOLD() {

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

	// stdoutPipe, err := c.Cmd.StdoutPipe()
	// if err != nil {
	// 	gotools.ShowErrorAndExit(fmt.Sprintf("Error al crear el pipe para la salida del programa: %s", err))
	// }

	// c.Scanner = bufio.NewScanner(stdoutPipe)

	// err = c.Cmd.Start()
	// if err != nil {
	// 	gotools.ShowErrorAndExit(fmt.Sprintf("Error al iniciar el programa: %s", err))
	// }

	time.Sleep(100 * time.Millisecond) // Esperar

}

func (d *Dev) StartProgram() {

	var err error
	// Cambiar al directorio "cmd" si existe
	cmdDir := "cmd"
	if _, err = os.Stat(cmdDir); err == nil {
		err = os.Chdir(cmdDir)
		if err != nil {
			gotools.ShowErrorAndExit(fmt.Sprintf("Error al cambiar al directorio '%s': %s", cmdDir, err))
		}
	}

	// BUILD AND RUN
	err = d.buildAndRun()
	if err != nil {
		gotools.ShowErrorAndExit(fmt.Sprintf("error al compilar e iniciar app: %s", err))
	}

}

func (d *Dev) Restart() error {

	// STOP
	err := d.stop()
	if err != nil {
		gotools.ShowErrorAndExit(fmt.Sprintf("error al cerrar app: %s", err))
	}

	// BUILD AND RUN
	err = d.buildAndRun()
	if err != nil {
		gotools.ShowErrorAndExit(fmt.Sprintf("error al compilar e iniciar app: %s", err))
	}

	d.Browser.Reload()

	return nil
}
