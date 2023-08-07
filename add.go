package godev

import (
	"bufio"
	"os"
	"os/exec"

	"github.com/cdvelop/compiler"
	"github.com/cdvelop/dev_browser"
	"github.com/cdvelop/watch_files"
)

// ir al directorio de componentes
// err := os.Chdir(c.components_dir)
// if err != nil {
// 	gotools.ShowErrorAndExit(fmt.Sprintf("Error al ir al directorio de componentes: %v %v", c.components_dir, err))
// }

func Add() *Dev {

	d := Dev{
		Browser:    dev_browser.Add(),
		WatchFiles: &watch_files.WatchFiles{},
		Compiler:   compiler.Config("compile_dir:cmd"),
		args:       []string{},
		Cmd:        &exec.Cmd{},
		Scanner:    &bufio.Scanner{},
		Interrupt:  make(chan os.Signal, 1),
	}

	d.WatchFiles = watch_files.Add(d, d, d, d.DirectoriesRegistered)

	return &d
}
