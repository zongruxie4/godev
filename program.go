package godev

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sync"
)

var (
	outputName    = "app"      // ej: app
	mainFilePath  string       // ej: test/app.go
	outPathApp    string       // eg : build/app.exe
	outputDir     = "build"    // ej: build
	run_arguments = []string{} // Inicializar sin argumentos
)

type Program struct {
	*exec.Cmd
	terminal *Terminal
}

func NewProgram(terminal *Terminal) *Program {

	p := &Program{
		Cmd:      &exec.Cmd{},
		terminal: terminal,
	}

	if err := p.programCheck(); err != nil {
		log.Fatal(err)
	}

	return p
}

func (h *Program) programCheck() error {

	if len(os.Args) < 2 {
		fmt.Println("Usage: godev <mainFilePath> [outputName] [outputDir]")
		fmt.Println("Parameters:")
		fmt.Println("  mainFilePath   : Path to main file (e.g., backend/main.go, server.go default: cmd/main.go)")
		fmt.Println("  outputName : Name of output executable (default: app)")
		fmt.Println("  outputDir  : Output directory (default: build)")
		os.Exit(1)
	}

	// Obtener el archivo principal a compilar
	mainFilePath = path.Join("cmd", "main.go") // Valor por defecto
	if len(os.Args) > 1 && os.Args[1] != "" {
		mainFilePath = os.Args[1]
	}

	if _, err := os.Stat(mainFilePath); errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Main file not found: %s", mainFilePath)
	}

	// Crear el directorio de salida si no existe
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatalf("No se pudo crear el directorio de salida: %v", err)
	}

	var exe_ext = ""
	if runtime.GOOS == "windows" {
		exe_ext = ".exe"
	}

	outPathApp = path.Join(outputDir, outputName+exe_ext)

	return nil
}

func (h *Program) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	// var this = errors.New("StartProgram")

	// if strings.Contains(h.main_file, "cmd") {
	// 	// Cambiar al directorio "cmd" si existe
	// 	cmdDir := "cmd"
	// 	if _, err := os.Stat(cmdDir); err == nil {
	// 		err := os.Chdir(cmdDir)
	// 		if err != nil {
	// 			PrintError(this, err, "al cambiar al directorio ", cmdDir)
	// 		}
	// 	}
	// }

	// BUILD AND RUN
	err := h.buildAndRun()
	if err != nil {
		h.terminal.MsgError("StartProgram ", err)
	}
}

func (h *Program) buildAndRun() error {
	var this = errors.New("buildAndRun")

	h.terminal.Msg(this, outputName, "...")

	os.Remove(outPathApp)

	// flags, err := ldflags.Add(
	// 	h.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// // sessionbackend.AddPrivateSecretKeySigning(),
	// )

	// var ldflags = `-X 'main.version=` + tag + `'`

	h.Cmd = exec.Command("go", "build", "-o", outPathApp, mainFilePath)
	// d.Cmd = exec.Command("go", "build", "-o", d.app_path, "main.go" )

	stderr, err := h.Cmd.StderrPipe()
	if err != nil {
		return errors.Join(this, err)
	}

	stdout, err := h.Cmd.StdoutPipe()
	if err != nil {
		return errors.Join(this, err)
	}

	err = h.Cmd.Start()
	if err != nil {
		return errors.Join(this, err)
	}

	io.Copy(os.Stdout, stdout)
	errBuf, _ := io.ReadAll(stderr)

	// Esperar
	err = h.Cmd.Wait()
	if err != nil {
		return errors.Join(this, errors.New(mainFilePath+" "+string(errBuf)), err)
	}

	return h.run()
}

// Construir el comando con argumentos dinámicos
// cmdArgs := append([]string{"go", "build", "-o", d.app_path, "main.go"}, os.Args...)
// d.Cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)

func (h *Program) run() error {

	h.Cmd = exec.Command(outPathApp)
	// h.Cmd = exec.Command("./"+d.app_path,h.main_file ,h.run_arguments...)

	stderr, err := h.Cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := h.Cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = h.Cmd.Start()
	if err != nil {
		return err
	}

	go io.Copy(h.terminal, stderr)
	go io.Copy(h.terminal, stdout)

	return nil
}

func (h *Program) Restart(event_name string) error {
	var this = errors.New("Restart")
	h.terminal.MsgWarning(this, "APP...", event_name)

	// STOP
	err := h.StopProgram()
	if err != nil {
		return errors.Join(this, err)

	}

	// BUILD AND RUN
	err = h.buildAndRun()
	if err != nil {
		return errors.Join(this, err)
	}

	return nil
}

func (h *Program) StopProgram() error {
	var this = errors.New("StopProgram")

	h.terminal.MsgWarning(this, "PID:", h.Cmd.Process.Pid)

	err := h.Cmd.Process.Kill()
	if err != nil {
		return errors.Join(this, err)
	}

	return nil
}
