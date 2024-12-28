package godev

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func (h *handler) StartProgram(wg *sync.WaitGroup) {
	defer wg.Done()
	var this = errors.New("StartProgram")

	if strings.Contains(h.main_file, "cmd") {
		// Cambiar al directorio "cmd" si existe
		cmdDir := "cmd"
		if _, err := os.Stat(cmdDir); err == nil {
			err := os.Chdir(cmdDir)
			if err != nil {
				PrintError(this, err, "al cambiar al directorio ", cmdDir)
			}
		}
	}

	// BUILD AND RUN
	err := h.buildAndRun()
	if err != nil {
		PrintError("StartProgram ", err)
	}
}

func (h *handler) buildAndRun() error {
	var this = errors.New("buildAndRun")
	PrintOK(fmt.Sprintf("Building and Running %s...\n", h.output_name))

	os.Remove(h.out_app_path)

	// flags, err := ldflags.Add(
	// 	h.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// // sessionbackend.AddPrivateSecretKeySigning(),
	// )

	// var ldflags = `-X 'main.version=` + tag + `'`

	h.Cmd = exec.Command("go", "build", "-o", h.out_app_path, h.main_file)
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
		errors.Join(this, errors.New(string(errBuf)), err)
	}

	return h.run()
}

// Construir el comando con argumentos dinámicos
// cmdArgs := append([]string{"go", "build", "-o", d.app_path, "main.go"}, os.Args...)
// d.Cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)

func (h *handler) run() error {

	h.Cmd = exec.Command(h.out_app_path)
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

	go io.Copy(h, stderr)
	go io.Copy(h, stdout)

	return nil
}

func (h *handler) Restart(event_name string) error {
	var this = errors.New("Restart")
	PrintWarning("Reiniciando APP...", event_name)

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

func (h *handler) StopProgram() error {
	var this = errors.New("StopProgram")

	PrintWarning(this, "PID:", h.Cmd.Process.Pid)

	err := h.Cmd.Process.Kill()
	if err != nil {
		return errors.Join(this, err)
	}

	return nil
}

// func (h *handler) writeToTerminal(message string, prefix string) {

// 	if message = strings.TrimSpace(message); message != "" {
// 		timestamp := time.Now().Format("15:04:05")
// 		formattedMsg := fmt.Sprintf("[%s][%s] %s", timestamp, prefix, message)

// 		if h.terminal != nil {
// 			h.terminal.messages = append(h.terminal.messages, formattedMsg)
// 			h.terminal.forceUpdate()
// 		}
// 	}
// }

func (h handler) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		// h.writeToTerminal(msg, "APP")
	}
	return len(p), nil
}
