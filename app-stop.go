package godev

import (
	"fmt"
	"os/exec"
	"syscall"
)

func (c *Dev) StopProgram() error {
	if c.Cmd == nil || c.Cmd.Process == nil {
		return nil
	}

	// Send a kill signal to the program
	err := c.Cmd.Process.Signal(syscall.SIGINT) // Primero, intenta enviar una señal SIGINT (Ctrl+C)
	if err != nil {
		// Si no se admite SIGINT, intenta con SIGTERM
		err = c.Cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			// Si tampoco se admite SIGTERM, utiliza Kill() para finalizar el proceso
			err = c.Cmd.Process.Kill()
			if err != nil {
				return fmt.Errorf("Failed to kill the program: %s\n", err)
			}
		}
	}

	// Wait for the program to finish
	err = c.Cmd.Wait()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok && exitErr.Sys().(syscall.WaitStatus).ExitStatus() != 1 {
			return fmt.Errorf("Program finished with error: %v\n", err)
		}
	}

	return nil
}
