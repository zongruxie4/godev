package godev

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
)

var (
	run_arguments = []string{} // Inicializar sin argumentos
)

type Program struct {
	*exec.Cmd
}

func (h *handler) NewProgram() {

	h.program = &Program{
		Cmd: &exec.Cmd{},
	}
	return
}

func (h *handler) ProgramStart(wg *sync.WaitGroup) {
	defer wg.Done()

	if len(os.Args) < 2 && !configFileFound {

		pathMainFile, err := findMainFile()
		if err != nil {
			h.terminal.MsgError("findMainFile ", err)
			h.showHelpExecProgram()
			return
		}
		config.MainFilePath = pathMainFile

		h.terminal.MsgOk("MainFile: " + pathMainFile)

	}

	// BUILD AND RUN
	err := h.buildAndRunProgram()
	if err != nil {
		h.terminal.MsgError("StartProgram ", err)
		return
	}

	// Esperar señal de cierre
	<-exitChan
}

func (h *handler) buildAndRunProgram() error {
	var this = errors.New("buildAndRun")

	h.terminal.Msg(this, config.AppName, "...")

	os.Remove(config.OutPathApp)

	// flags, err := ldflags.Add(
	// 	h.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// 	h.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// // sessionbackend.AddPrivateSecretKeySigning(),
	// )

	// var ldflags = `-X 'main.version=` + tag + `'`

	h.program.Cmd = exec.Command("go", "build", "-o", config.OutPathApp, config.MainFilePath)
	// d.Cmd = exec.Command("go", "build", "-o", d.app_path, "main.go" )

	stderr, err := h.program.Cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := h.program.Cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = h.program.Cmd.Start()
	if err != nil {
		return err
	}

	go io.Copy(h.terminal, stderr)
	go io.Copy(h.terminal, stdout)

	return nil
}

func (h *handler) showHelpExecProgram() {
	h.terminal.MsgInfo(`Usage for build app without config file eg: godev <MainFilePath> [AppName] [OutputDir]`)
	h.terminal.MsgInfo(`Parameters:`)
	h.terminal.MsgInfo(`MainFilePath : Path to main file eg: backend/main.go, server.go (default: cmd/main.go)`)
	h.terminal.MsgInfo(`AppName      : Name of output executable eg: miAppName, server (default: app)`)
	h.terminal.MsgInfo(`OutputDir    : Output directory eg: dist/build (default: build)`)
}

// Construir el comando con argumentos dinámicos
// cmdArgs := append([]string{"go", "build", "-o", d.app_path, "main.go"}, os.Args...)
// d.Cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)

func (h *handler) runProgram() error {

	h.program.Cmd = exec.Command(config.OutPathApp)
	// h.Cmd = exec.Command("./"+d.app_path,h.main_file ,h.run_arguments...)

	stderr, err := h.program.Cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := h.program.Cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = h.program.Cmd.Start()
	if err != nil {
		return err
	}

	go io.Copy(h.terminal, stderr)
	go io.Copy(h.terminal, stdout)

	return nil
}

func (h *handler) RestartProgram(event_name string) error {
	var this = errors.New("Restart")
	h.terminal.MsgWarning(this, "APP...", event_name)

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

func (h *handler) StopProgram() error {
	var this = errors.New("StopProgram")

	h.terminal.MsgWarning(this, "PID:", h.program.Cmd.Process.Pid)

	err := h.program.Cmd.Process.Kill()
	if err != nil {
		return errors.Join(this, err)
	}

	return nil
}
