package godev

import (
	"fmt"
	"os"

	. "github.com/cdvelop/output"
)

func (d *Dev) StartProgram() {

	// Cambiar al directorio "cmd" si existe
	cmdDir := "cmd"
	if _, err := os.Stat(cmdDir); err == nil {
		err = os.Chdir(cmdDir)
		if err != nil {
			ShowErrorAndExit(fmt.Sprintf("al cambiar al directorio '%s': %s", cmdDir, err))
		}
	}

	// BUILD AND RUN
	err := d.buildAndRun()
	if err != nil {
		ShowErrorAndExit(fmt.Sprintf("al compilar e iniciar app: %s", err))
	}

}

func (d *Dev) Restart() error {

	// STOP
	err := d.StopProgram()
	if err != nil {
		PrintError(fmt.Sprintf("al cerrar app: %s", err))
	}

	// BUILD AND RUN
	err = d.buildAndRun()
	if err != nil {
		ShowErrorAndExit(fmt.Sprintf("al compilar e iniciar app: %s", err))
	}

	return nil
}
