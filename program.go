package godev

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func (h *handler) StartProgram() {

	// BUILD AND RUN
	err := h.buildAndRun()
	if err != nil {
		PrintError("StartProgram " + err.Error())
	}

}

func (h *handler) Restart(event_name string) error {
	var this = errors.New("Restart error")
	fmt.Println("Restarting APP..." + event_name)

	// STOP
	err := h.StopProgram()
	if err != nil {
		return errors.Join(this, errors.New("when closing app"), err)
	}

	// BUILD AND RUN
	err = h.buildAndRun()
	if err != nil {
		return errors.Join(this, errors.New("when building and starting app"), err)
	}

	return nil
}

func (h *handler) buildAndRun() error {
	var this = errors.New("buildAndRun")
	PrintWarning(fmt.Sprintf("Building and Running %s...\n", h.app_path))

	err := os.Remove(h.app_path)
	if err != nil {
		return errors.Join(this, err)
	}
	// flags, err := ldflags.Add(
	// 	d.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// // sessionbackend.AddPrivateSecretKeySigning(),
	// )

	// var ldflags = `-X 'main.version=` + tag + `'`

	h.Cmd = exec.Command("go", "build", "-o", h.app_path, "-ldflags", "main.go")
	// h.Cmd = exec.Command("go", "build", "-o", h.app_path, "-ldflags", flags, "main.go")
	// d.Cmd = exec.Command("go", "build", "-o", d.app_path, "main.go" )

	stderr, er := h.Cmd.StderrPipe()
	if er != nil {
		return errors.Join(this, err)
	}

	stdout, er := h.Cmd.StdoutPipe()
	if er != nil {
		return errors.Join(this, err)
	}

	er = h.Cmd.Start()
	if er != nil {
		return errors.Join(this, err)
	}

	io.Copy(os.Stdout, stdout)
	errBuf, _ := io.ReadAll(stderr)

	// Esperar
	er = h.Cmd.Wait()
	if er != nil {
		return errors.Join(this, errors.New(string(errBuf)), err)
	}

	return h.run()
}

// Construir el comando con argumentos dinámicos
// cmdArgs := append([]string{"go", "build", "-o", d.app_path, "main.go"}, os.Args...)
// d.Cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)

func (h *handler) run() error {
	var this = errors.New("run")

	h.Cmd = exec.Command("./"+h.app_path, h.run_arguments...)

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

	go io.Copy(h, stderr)
	go io.Copy(h, stdout)

	return nil
}

func (h handler) Write(p []byte) (n int, err error) {
	h.ProgramMessages <- string(p)
	// fmt.Println(string(p))
	return len(p), nil
}

func (h *handler) StopProgram() error {

	pid := h.Cmd.Process.Pid

	PrintWarning(fmt.Sprintf("stop app PID %d\n", pid))

	return h.Cmd.Process.Kill()
}
