package godev

import (
	"fmt"
	"os"
)

func showErrorAndExit(errorMessage string) {
	fmt.Println("Error: " + errorMessage)
	fmt.Println("")
	fmt.Println("Presione cualquier tecla para salir...")
	fmt.Scanln()
	os.Exit(1)
}
