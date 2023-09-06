package godev

import (
	"os"
	"os/exec"

	"github.com/cdvelop/compiler"
	"github.com/cdvelop/dev_browser"
	"github.com/cdvelop/watch_files"
)

func Add() *Dev {

	d := Dev{
		app_path:               "app.exe",
		Browser:                dev_browser.Add(),
		WatchFiles:             &watch_files.WatchFiles{},
		Compiler:               compiler.Config("compile_dir:cmd"),
		Cmd:                    &exec.Cmd{},
		Interrupt:              make(chan os.Signal, 1),
		ProgramStartedMessages: make(chan string),
	}

	d.WatchFiles = watch_files.Add(d, d, &d, d.DirectoriesRegistered, d.Compiler.ThemeDir())

	return &d
}
