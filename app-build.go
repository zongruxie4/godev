package godev

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func (d *Dev) buildAndRun() error {
	fmt.Printf("Building and Running %s...\n", d.app_path)

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
		return fmt.Errorf("%v %v\n", string(errBuf), err)
	}

	return d.run()
}

func (d *Dev) run() error {

	d.Cmd = exec.Command("./" + d.app_path)

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
	fmt.Println(string(p))

	return len(p), nil
}

func (d *Dev) stop() error {

	pid := d.Cmd.Process.Pid
	fmt.Printf("stop app Killing PID %d\n", pid)

	err := d.Cmd.Process.Kill()
	if err != nil {
		return err
	}

	return nil
}

func (c *Dev) ProcessProgramOutputOLD(ctx context.Context) {

	// for c.Scanner.Scan() {
	// line := c.Scanner.Text()

	// switch {
	// case strings.Contains(line, "restart_app"):
	// 	c.StopProgram()
	// 	c.StartProgram()
	// 	c.Browser.Reload()

	// case strings.Contains(line, "reload_browser"):
	// 	c.Browser.Reload()

	// case strings.Contains(line, "module:"):

	// var module string
	// ExtractTwoPointArgument(line, &module)

	// for _, m := range modules {
	// fmt.Println("MODULO RECIBIDO:", module)
	// }

	// default:

	// fmt.Println(line)

	// }
	// }

}
