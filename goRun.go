package godev

import (
	"fmt"
	"io"
	"os/exec"
)

type GoRunConfig struct {
	ExecProgramPath string // eg: "server/main.exe"
	RunArguments    func() []string
	ExitChan        chan bool
	Writer          io.Writer
}

type GoRun struct {
	*GoRunConfig
	Cmd       *exec.Cmd
	IsRunning bool
}

func NewGoRun(c *GoRunConfig) *GoRun {
	return &GoRun{
		GoRunConfig: c,
		Cmd:         &exec.Cmd{},
		IsRunning:   false,
	}
}

func (h *GoRun) RunProgram() error {

	runArgs := []string{}

	if h.RunArguments != nil {
		runArgs = h.RunArguments()
	}

	h.Cmd = exec.Command(h.ExecProgramPath, runArgs...)

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
	h.IsRunning = true

	done := make(chan struct{})

	go io.Copy(h.Writer, stderr)
	go io.Copy(h.Writer, stdout)

	go func() {
		select {
		case <-h.ExitChan:
			// h.Print("Received exit signal, stopping application...")
			h.StopProgram()
			close(done)
		case <-done:
			// finish goroutine
		}
	}()

	go func() {
		err := h.Cmd.Wait()
		if err != nil {
			fmt.Fprintf(h.Writer, "App: %v closed with error: %v\n", h.ExecProgramPath, err)
		} else {
			fmt.Fprintf(h.Writer, "App: %v closed successfully\n", h.ExecProgramPath)
		}
		close(done)
	}()

	return nil
}

func (h *GoRun) StopProgram() error {
	if !h.IsRunning || h.Cmd.Process == nil {
		h.IsRunning = false
		return nil
	}

	// First try to find if process exists
	if h.Cmd.ProcessState != nil && h.Cmd.ProcessState.Exited() {
		h.IsRunning = false
		return nil
	}

	h.IsRunning = false
	return h.Cmd.Process.Kill()
}
