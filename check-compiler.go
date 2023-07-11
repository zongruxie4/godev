package godev

import (
	"fmt"

	"github.com/cdvelop/gotools"
)

func (u ui) compilerCheck() {

	u.BuildHTML()

	err := gotools.FindFilesWithNonZeroSize(BUILT_FOLDER, []string{"style.css", "main.js"})
	if err != nil {
		fmt.Println(err, "... recompilando proyecto archivos: html,css,js,wasm ...")

		u.BuildJS()

		u.BuildCSS()

		u.BuildWASM()

	}
}
