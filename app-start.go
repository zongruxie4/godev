package godev

import (
	"fmt"
	"os"

	. "github.com/cdvelop/output"
)

func (d *Dev) StartProgram() {

	// Cambiar al directorio "cmd" si existe
	cmdDir := "cmd"
	if _, er := os.Stat(cmdDir); er == nil {
		e := os.Chdir(cmdDir)
		if e != nil {
			PrintError("StartProgram error " + er.Error() + " al cambiar al directorio '" + cmdDir + "': " + e.Error())
		}
	}

	// BUILD AND RUN
	err := d.buildAndRun()
	if err != "" {
		PrintError("StartProgram " + err)
	}

}

func (d *Dev) Restart(event_name string) (err string) {
	const this = "Restart error "
	fmt.Println("Reiniciando APP..." + event_name)

	// STOP
	err = d.StopProgram()
	if err != "" {
		return this + "al cerrar app: " + err

	}

	// BUILD AND RUN
	err = d.buildAndRun()
	if err != "" {
		return this + "al compilar e iniciar app: " + err
	}

	return
}
