package godev

import (
	"fmt"
	"os"
)

func ShowErrorAndExit(errorMessage string) {
	fmt.Println("Error: " + errorMessage)
	fmt.Println("")
	fmt.Println("Presiona enter para salir...")
	fmt.Scanln()
	os.Exit(1)
}
