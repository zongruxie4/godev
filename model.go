package godev

import (
	"os"
	"os/exec"

	"github.com/cdvelop/compiler"
	"github.com/cdvelop/dev_browser"
	"github.com/cdvelop/watch_files"
)

type Dev struct {
	app_path string //ej: app.exe

	*dev_browser.Browser
	*watch_files.WatchFiles
	*compiler.Compiler

	*exec.Cmd

	// Scanner   *bufio.Scanner
	Interrupt chan os.Signal

	ProgramStartedMessages chan string

	test_argument          string
	dev_argument           string
	cache_browser_argument string
}
