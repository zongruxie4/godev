package godev

import (
	"errors"
	"io"
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
	return p
}

func (h *Program) programFileOK() bool {

	if len(os.Args) < 2 {
		h.terminal.MsgInfo(`Usage for build app without godev.yml config file eg: godev <mainFilePath> [outputName] [outputDir]`)
		h.terminal.MsgInfo(`Parameters:`)
		h.terminal.MsgInfo(`mainFilePath : Path to main file eg: backend/main.go, server.go (default: cmd/main.go)`)
		h.terminal.MsgInfo(`outputName   : Name of output executable eg: miAppName, server (default: app)`)
		h.terminal.MsgInfo(`outputDir    : Output directory eg: dist/build (default: build)`)

		return false
	}

	// Obtener el archivo principal a compilar
	mainFilePath = path.Join("cmd", "main.go") // Valor por defecto
	if len(os.Args) > 1 && os.Args[1] != "" {
		mainFilePath = os.Args[1]
	}

	if _, err := os.Stat(mainFilePath); errors.Is(err, os.ErrNotExist) {
		h.terminal.MsgError("Main file not found: %s", mainFilePath)
		return false
	}

	// Crear el directorio de salida si no existe
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		h.terminal.MsgError("No se pudo crear el directorio de salida:", err)
		return false
	}

	var exe_ext = ""
	if runtime.GOOS == "windows" {
		exe_ext = ".exe"
	}

	outPathApp = path.Join(outputDir, outputName+exe_ext)

	return true
}

func (h *Program) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	if !h.programFileOK() {
		return
	}

	// BUILD AND RUN
	err := h.buildAndRun()
	if err != nil {
		h.terminal.MsgError("StartProgram ", err)
		return
	}
}

func (h *Program) buildAndRun() error {
	var this = errors.New("buildAndRun")

	h.terminal.Msg(this, outputName, "...")

	os.Remove(outPathApp)

	// flags, err := ldflags.Add(
	// 	h.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// 	h.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// // sessionbackend.AddPrivateSecretKeySigning(),
	// )

	// var ldflags = `-X 'main.version=` + tag + `'`

	h.Cmd = exec.Command("go", "build", "-o", outPathApp, mainFilePath)
	// d.Cmd = exec.Command("go", "build", "-o", d.app_path, "main.go" )

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
