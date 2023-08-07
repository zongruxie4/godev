package godev

import (
	"fmt"
	"strings"
	"sync"
)

func (c *Dev) ProcessProgramOutput(wg *sync.WaitGroup) {
	defer wg.Done()

	for c.Scanner.Scan() {
		line := c.Scanner.Text()

		switch {
		case strings.Contains(line, "restart_app"):
			c.StopProgram()
			c.StartProgram()
			c.Browser.Reload()

		case strings.Contains(line, "reload_browser"):
			c.Browser.Reload()

		// case strings.Contains(line, "module:"):

		// var module string
		// ExtractTwoPointArgument(line, &module)

		// for _, m := range modules {
		// fmt.Println("MODULO RECIBIDO:", module)
		// }

		default:

			fmt.Println(line)

		}
	}

}
