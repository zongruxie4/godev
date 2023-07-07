package godev

import (
	"fmt"

	"github.com/cdvelop/gotools"
)

func (u ui) compilerCheck() {

	err := gotools.FindFilesWithNonZeroSize(BuiltFolder, []string{"index.html", "style.css", "main.js"})
	if err != nil {
		fmt.Println("ARCHIVOS NO ENCONTRADOS: ", err)
	}
}
