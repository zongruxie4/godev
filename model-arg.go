package godev

import (
	"bufio"
	"context"
	"os"
	"os/exec"
)

type Args struct {
	path     string // ej /index.html, /
	port     string //8080
	domain   string // localhost
	protocol string // ej https,

	args []string
	*exec.Cmd

	Scanner   *bufio.Scanner
	Interrupt chan os.Signal

	with     string //browser option
	height   string //browser option
	position string //browser option

	context.Context
	context.CancelFunc
}
