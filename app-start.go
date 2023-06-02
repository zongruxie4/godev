package godev

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
)

func (a *Args) StartProgramOLD() *exec.Cmd {
	cmd := exec.Command("go", "run", "main.go") // Replace "program_name" with the actual program name and provide any required arguments

	// Set up the appropriate output and error streams
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fmt.Printf("Failed to start the program: %s\n", err)
		return nil
	}

	fmt.Println("Program started successfully.")

	return cmd
}

// func (a *Args) FirstStartProgram(wg *sync.WaitGroup) {
// 	defer wg.Done()

// 	a.StartProgram()

// 	// Esperar 1 segundo
// 	time.Sleep(1 * time.Second)
// }

func (a *Args) StartProgram() {

	a.Cmd = exec.Command("go", "run", "main.go", "port="+a.Port)

	stdoutPipe, err := a.Cmd.StdoutPipe()
	if err != nil {
		a.ShowErrorAndExit(fmt.Sprintf("Error al crear el pipe para la salida del programa: %s", err))
	}

	a.Scanner = bufio.NewScanner(stdoutPipe)

	err = a.Cmd.Start()
	if err != nil {
		a.ShowErrorAndExit(fmt.Sprintf("Error al iniciar el programa: %s", err))
	}

	// fmt.Println("Programa iniciado exitosamente.")
}
