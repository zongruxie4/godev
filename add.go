package godev

import (
	"os"
	"os/exec"

	"github.com/cdvelop/compiler"
	"github.com/cdvelop/dev_browser"
	"github.com/cdvelop/strings"
	"github.com/cdvelop/token"
	"github.com/cdvelop/watch_files"
)

func Add() *Dev {
	d := &Dev{
		app_path:               "app.exe",
		Browser:                dev_browser.Add(),
		WatchFiles:             &watch_files.WatchFiles{},
		Cmd:                    &exec.Cmd{},
		Interrupt:              make(chan os.Signal, 1),
		ProgramStartedMessages: make(chan string),
		TwoKeys:                &token.TwoKeys{},
	}

	d.app_path = d.AppFileName()

	var test_suffix string

	cache_browser_argument := "no-cache"

	for _, v := range os.Args {
		if strings.Contains(v, "test:") != 0 {
			d.run_arguments = append(d.run_arguments, v)
			test_suffix = "test:"
		}
		if v == "dev" {
			d.run_arguments = append(d.run_arguments, v)
		}
		if v == "cache" {
			cache_browser_argument = v
		}
	}

	d.Compiler = compiler.Add(&compiler.Config{
		AppInfo:             d,
		TwoPublicKeyAdapter: d.TwoKeys,
	}, test_suffix, "compile_dir:cmd")

	d.run_arguments = append(d.run_arguments, cache_browser_argument)

	d.WatchFiles = watch_files.Add(d, d, d, d.DirectoriesRegistered, d.Compiler.ThemeDir())

	return d
}
