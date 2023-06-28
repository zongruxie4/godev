package godev

import (
	"fmt"
	"os"
)

func ShowErrorAndExit(errorMessage string) {
	fmt.Println("Error: " + errorMessage)
	fmt.Println("")
	fmt.Println("Presione cualquier tecla para salir...")
	fmt.Scanln()
	os.Exit(1)
}
