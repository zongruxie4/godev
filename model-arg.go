package godev

import (
	"bufio"
	"context"
	"os"
	"os/exec"
)

type Args struct {
	browser_path string // ej /index.html
	app_port     bool
	args         []string
	*exec.Cmd

	Scanner   *bufio.Scanner
	Interrupt chan os.Signal

	with     string //browser option
	height   string //browser option
	position string //browser option

	context.Context
	context.CancelFunc
}
