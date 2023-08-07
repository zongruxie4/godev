package godev

import (
	"bufio"
	"os"
	"os/exec"

	"github.com/cdvelop/compiler"
	"github.com/cdvelop/dev_browser"
	"github.com/cdvelop/watch_files"
)

type Dev struct {
	*dev_browser.Browser
	*watch_files.WatchFiles
	*compiler.Compiler

	args []string
	*exec.Cmd

	Scanner   *bufio.Scanner
	Interrupt chan os.Signal
}
