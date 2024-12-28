package godev

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type handler struct {
	app_path string //ej: build/app.exe

	*exec.Cmd
	// Scanner   *bufio.Scanner
	// Interrupt              chan os.Signal
	ProgramMessages chan string
	run_arguments   []string

	terminal *Terminal
	tea      *tea.Program
}

func GodevStart() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: godev <main_file> [output_name] [output_dir]")
		fmt.Println("Parameters:")
		fmt.Println("  main_file   : Path to main file (e.g., backend/main.go, server.go default: cmd/main.go)")
		fmt.Println("  output_name : Name of output executable (default: app)")
		fmt.Println("  output_dir  : Output directory (default: build)")
		os.Exit(1)
	}

	mainFile := "cmd/main.go"
	outputName := "app"
	outputDir := "build"

	if len(os.Args) > 1 && os.Args[1] != "" {
		mainFile = os.Args[1]
	}

	if len(os.Args) > 2 && os.Args[2] != "" {
		outputName = os.Args[2]
	}

	if len(os.Args) > 3 && os.Args[3] != "" {
		outputDir = os.Args[3]
	}

	if _, err := os.Stat(mainFile); errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Main file not found: %s", mainFile)
	}

	var exe_ext = ""
	if runtime.GOOS == "windows" {
		exe_ext = ".exe"
	}

	h := &handler{
		app_path: path.Join(outputDir, outputName+exe_ext),
		Cmd:      &exec.Cmd{},
		// Interrupt:              make(chan os.Signal, 1),
		ProgramMessages: make(chan string),
		// terminal:               NewTerminal(),
	}

	h.NewTerminal()

	// Cree un canal para recibir señales de interrupción
	// signal.Notify(h.Interrupt, os.Interrupt, syscall.SIGTERM)

	// var wg sync.WaitGroup
	// wg.Add(2)

	// go h.StartProgram()

	var app_started bool

	for {

		select {

		case message := <-h.ProgramMessages:

			if strings.Contains(strings.ToLower(message), "err") {
				fmt.Println(message)
			} else {
				fmt.Println(message)

				// go d.DevBrowserSTART(&wg)

				// go d.DevFileWatcherSTART(&wg)

				if !app_started {

					// var wg sync.WaitGroup
					// wg.Add(2)

					// go h.DevBrowserSTART(&wg)

					// go h.DevFileWatcherSTART(&wg)

					app_started = true
				}
			}

			// case <-h.Interrupt:
			// 	// Detenga el navegador y cierre la aplicación cuando se recibe una señal de interrupción
			// 	// if er := chromedp.Cancel(d.Context); er != nil {
			// 	// 	log.Println("al cerrar browser: " + er.Error())
			// 	// }
			// 	err := h.StopProgram()
			// 	if err != nil {
			// 		log.Println("al detener app: " + err.Error())
			// 	}

			// 	os.Exit(0)
		}

	}

}
