package godev

import (
	"fmt"

	"github.com/cdvelop/gotools"
)

func (u ui) compilerCheck() {

	err := gotools.FindFilesWithNonZeroSize(BUILT_FOLDER, []string{"index.html", "style.css", "main.js"})
	if err != nil {
		fmt.Println(err, "... recompilando proyecto archivos: html,css,js,wasm ...")

		u.BuildHTML()

		u.BuildJS()

		u.BuildCSS()

		u.BuildWASM()

	}
}
