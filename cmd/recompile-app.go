package main

import (
	"os"
	"os/exec"
)

func recompileApp() error {
	cmd := exec.Command("go", "build", "-o", getAppName())
	cmd.Dir = getAppDirectory()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
