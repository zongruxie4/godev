package godev

import (
	"bufio"
	"context"
	"os"
	"os/exec"
)

type Args struct {
	Path string // ej http://localhost:8080/index.html
	Port string //ej: 8080
	*exec.Cmd

	Scanner   *bufio.Scanner
	Reload    chan bool
	Interrupt chan os.Signal

	context.Context
	context.CancelFunc
}
