package main

import (
	"os"
	"os/exec"
)

func restartApp() error {
	err := killApp()
	if err != nil {
		return err
	}

	cmd := exec.Command("./" + getAppName())
	cmd.Dir = getAppDirectory()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return err
	}

	appProcess = cmd.Process // Actualiza la variable appProcess con el nuevo proceso de la aplicación reiniciada

	return nil
}
