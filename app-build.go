package godev

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	. "github.com/cdvelop/output"
)

func (d *Dev) buildAndRun() error {

	PrintWarning(fmt.Sprintf("Building and Running %s...\n", d.app_path))

	os.Remove(d.app_path)

	d.Cmd = exec.Command("go", "build", "-o", d.app_path, "main.go")

	stderr, err := d.Cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := d.Cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = d.Cmd.Start()
	if err != nil {
		return err
	}

	io.Copy(os.Stdout, stdout)
	errBuf, _ := io.ReadAll(stderr)

	// Esperar
	err = d.Cmd.Wait()
	if err != nil {
		return fmt.Errorf("%v %v", string(errBuf), err)
	}

	return d.run()
}

func (d *Dev) run() error {

	d.Cmd = exec.Command("./"+d.app_path, "dev")

	stderr, err := d.Cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdout, err := d.Cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = d.Cmd.Start()
	if err != nil {
		return err
	}

	go io.Copy(d, stderr)
	go io.Copy(d, stdout)

	return nil
}

func (d Dev) Write(p []byte) (n int, err error) {
	d.ProgramStartedMessages <- string(p)
	// fmt.Println(string(p))
	return len(p), nil
}

func (d *Dev) StopProgram() error {

	pid := d.Cmd.Process.Pid

	PrintWarning(fmt.Sprintf("stop app PID %d\n", pid))

	err := d.Cmd.Process.Kill()
	if err != nil {
		return err
	}

	return nil
}
