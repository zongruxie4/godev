package godev

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type handler struct {
	main_file    string // ej: test/app.go
	output_name  string // ej: app
	output_dir   string // ej: build
	out_app_path string // ej: built/app.exe

	*exec.Cmd
	// Scanner   *bufio.Scanner
	// Interrupt              chan os.Signal
	run_arguments []string

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

	// Obtener el archivo principal a compilar
	mainFile := path.Join("cmd", "main.go") // Valor por defecto
	if len(os.Args) > 1 && os.Args[1] != "" {
		mainFile = os.Args[1]
	}

	// Verificar si el archivo existe
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		log.Fatalf("Archivo principal no encontrado: %s", mainFile)
	}

	outputName := "app"
	outputDir := "build"

	// Crear el directorio de salida si no existe
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatalf("No se pudo crear el directorio de salida: %v", err)
	}

	if _, err := os.Stat(mainFile); errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Main file not found: %s", mainFile)
	}

	var exe_ext = ""
	if runtime.GOOS == "windows" {
		exe_ext = ".exe"
	}

	h := &handler{
		main_file:     mainFile,
		output_name:   outputName + exe_ext,
		output_dir:    outputDir,
		out_app_path:  path.Join(outputDir, outputName+exe_ext),
		Cmd:           &exec.Cmd{},
		run_arguments: []string{}, // Inicializar sin argumentos
	}

	h.NewTerminal()

	// Cree un canal para recibir señales de interrupción
	// signal.Notify(h.Interrupt, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(2)

	// Iniciar la terminal en una goroutine
	go h.RunTerminal(&wg)

	// Esperar a que la terminal esté lista
	// Iniciar el programa
	go h.StartProgram(&wg)
	wg.Wait()

	// Mantener el programa activo
	select {}

}
