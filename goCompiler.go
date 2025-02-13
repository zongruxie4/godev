package godev

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
)

type GoCompiler struct {
	*GoCompilerConfig
	*exec.Cmd
}

type GoCompilerConfig struct {
	MainFilePath   func() string         //ej: web/main.server.go, cmd/main.go
	OutPathAppName func() string         // eg: web/miApp.exe, cmd/build/miApp
	RunArguments   func() []string       // argumentos de arranque eg: -p 10000
	Print          func(messages ...any) // eg: fmt.Println
	Writer         io.Writer             // escritor de mensajes de destino eg:io.Writer, os.Stdout
	ExitChan       chan bool             // Canal global para señalizar el cierre
}

func NewGoCompiler(c *GoCompilerConfig) *GoCompiler {

	g := &GoCompiler{
		GoCompilerConfig: c,
		Cmd:              &exec.Cmd{},
	}

	return g
}

// eg: miApp.exe
func (h *GoCompiler) UnchangeableOutputFileNames() []string {

	fileName, err := GetFileName(h.OutPathAppName())
	if err != nil {
		h.Print("GoCompiler UnchangeableOutputFileNames", err)
		return []string{}
	}

	return []string{
		fileName,
	}
}

func (h *GoCompiler) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	h.Print("GoCompiler Start", h.MainFilePath())

	// BUILD AND RUN
	err := h.buildAndRunProgram()
	if err != nil {
		h.Print("GoCompiler Start", err)
		return
	}

	// Esperar señal de cierre
	<-h.ExitChan
}

func (h *GoCompiler) buildAndRunProgram() error {
	var this = errors.New("buildAndRun")

	os.Remove(h.OutPathAppName())

	// flags, err := ldflags.Add(
	// 	h.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// 	h.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// // sessionbackend.AddPrivateSecretKeySigning(),
	// )

	// var ldflags = `-X 'main.version=` + tag + `'`

	h.Cmd = exec.Command("go", "build", "-o", h.OutPathAppName(), h.MainFilePath())
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

	go io.Copy(h.Writer, stderr)
	go io.Copy(h.Writer, stdout)

	return nil
}

func (h *GoCompiler) showHelpExecProgram() {
	h.Print(`Usage for build app without config file eg: godev <MainFilePath> [AppName] [WebFilesFolder]`)
	h.Print(`Parameters:`)
	h.Print(`MainFilePath : Path to main file eg: backend/main.go, server.go (default: cmd/main.go)`)
	h.Print(`AppName      : Name of output executable eg: miAppName, server (default: app)`)
	h.Print(`WebFilesFolder    : Output directory eg: dist/build (default: build)`)
}

// Construir el comando con argumentos dinámicos
// cmdArgs := append([]string{"go", "build", "-o", d.app_path, "main.go"}, os.Args...)
// d.Cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)

func (h *GoCompiler) runProgram() error {

	h.Cmd = exec.Command(h.OutPathAppName(), h.RunArguments()...)
	// h.Cmd = exec.Command("./"+d.app_path,h.main_file ,h.RunArguments...)

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

	go io.Copy(h.Writer, stderr)
	go io.Copy(h.Writer, stdout)

	return nil
}

func (h *GoCompiler) RestartProgram(event_name string) error {
	var this = errors.New("Restart")
	h.Print(this, "APP...", event_name)

	// STOP
	err := h.StopProgram()
	if err != nil {
		return errors.Join(this, err)

	}

	// BUILD AND RUN
	err = h.buildAndRunProgram()
	if err != nil {
		return errors.Join(this, err)
	}

	return nil
}

func (h *GoCompiler) StopProgram() error {
	var this = errors.New("StopProgram")

	h.Print(this, "PID:", h.Cmd.Process.Pid)

	err := h.Cmd.Process.Kill()
	if err != nil {
		return errors.Join(this, err)
	}

	return nil
}
