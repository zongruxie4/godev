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
			PrintError(fmt.Sprintf("al cambiar al directorio '%s': %s", cmdDir, err))
		}
	}

	// BUILD AND RUN
	err := d.buildAndRun()
	if err != nil {
		PrintError(fmt.Sprintf("al compilar e iniciar app: %s", err))
	}

}

func (d *Dev) Restart(event_name string) error {
	fmt.Println("Reiniciando APP..." + event_name)

	// STOP
	err := d.StopProgram()
	if err != nil {
		return fmt.Errorf("al cerrar app: %s", err)

	}

	// BUILD AND RUN
	err = d.buildAndRun()
	if err != nil {
		return fmt.Errorf("al compilar e iniciar app: %s", err)
	}

	return nil
}
