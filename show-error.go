package godev

import (
	"fmt"
	"os"
)

func (Args) ShowErrorAndExit(errorMessage string) {
	fmt.Println(errorMessage)
	fmt.Println("")
	fmt.Println("Presione cualquier tecla para salir...")
	fmt.Scanln()
	os.Exit(1)
}
