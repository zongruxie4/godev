package godev

import (
	"fmt"
	"os"

	"github.com/cdvelop/gotools"
)

func (d *Dev) StartProgram() {

	// Cambiar al directorio "cmd" si existe
	cmdDir := "cmd"
	if _, err := os.Stat(cmdDir); err == nil {
		err = os.Chdir(cmdDir)
		if err != nil {
			gotools.ShowErrorAndExit(fmt.Sprintf("Error al cambiar al directorio '%s': %s", cmdDir, err))
		}
	}

	// BUILD AND RUN
	err := d.buildAndRun()
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
