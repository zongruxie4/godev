package godev

import (
	"fmt"
	"os/exec"
	"syscall"
)

func (a *Args) StopProgram() {
	if a.Cmd == nil || a.Cmd.Process == nil {
		return
	}

	// Send a kill signal to the program
	err := a.Cmd.Process.Signal(syscall.SIGINT) // Primero, intenta enviar una señal SIGINT (Ctrl+C)
	if err != nil {
		// Si no se admite SIGINT, intenta con SIGTERM
		err = a.Cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			// Si tampoco se admite SIGTERM, utiliza Kill() para finalizar el proceso
			err = a.Cmd.Process.Kill()
			if err != nil {
				fmt.Printf("Failed to kill the program: %s\n", err)
				return
			}
		}
	}

	// Wait for the program to finish
	err = a.Cmd.Wait()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok && exitErr.Sys().(syscall.WaitStatus).ExitStatus() != 1 {
			fmt.Printf("Program finished with error: %v\n", err)
		}
		return
	}

	fmt.Println("Program stopped successfully.")
}
