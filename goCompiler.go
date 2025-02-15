package godev

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
)

type GoCompiler struct {
	*GoCompilerConfig
	*exec.Cmd

	outFileName     string // eg win: miApp.exe, unix: miApp
	outTempFileName string // eg win: miApp_temp.exe, unix: miApp_temp
}

type GoCompilerConfig struct {
	MainFilePath func() string   //ej: web/main.server.go, cmd/main.go
	AppName      func() string   // eg: miApp, miFirsWebApp
	RunArguments func() []string // argumentos de arranque eg: -p 10000
	//for dynamic ldflags eg: LDFlags: func() []string {
	//     return []string{
	//         `-X 'main.version=v1.0.0'`,
	//         `-X 'main.key=value'`,
	//     }
	LDFlags   func() []string
	OutFolder func() string         // eg: build, dist web
	Print     func(messages ...any) // eg: fmt.Println
	ExitChan  chan bool             // Canal global para señalizar el cierre
}

func NewGoCompiler(c *GoCompilerConfig) *GoCompiler {

	var exe_ext = ""
	if runtime.GOOS == "windows" {
		exe_ext = ".exe"
	}

	g := &GoCompiler{
		GoCompilerConfig: c,
		Cmd:              &exec.Cmd{},
		outFileName:      c.AppName() + exe_ext,
		outTempFileName:  c.AppName() + "_temp" + exe_ext,
	}

	return g
}

// eg: miApp.exe, miApp_temp.exe
func (h *GoCompiler) UnchangeableOutputFileNames() []string {
	return []string{
		h.outFileName,
		h.outTempFileName,
	}
}

func (h *GoCompiler) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	h.Print("GoCompiler Start", h.MainFilePath())

	// BUILD
	err := h.BuildProgram()
	if err != nil {
		h.Print("GoCompiler Start", err)
		return
	}

	// RUN
	err = h.RunProgram()
	if err != nil {
		h.Print("GoCompiler Start", err)
		return
	}

	// Esperar señal de cierre
	<-h.ExitChan
}

func (h *GoCompiler) BuildProgram() error {
	var this = errors.New("BuildProgram")

	buildArgs := []string{"build"}

	// Add multiple ldflags if provided
	if h.LDFlags != nil {
		flags := h.LDFlags()
		if len(flags) > 0 {
			ldflags := "-ldflags=" + strings.Join(flags, " ")
			buildArgs = append(buildArgs, ldflags)
		}
	}

	buildArgs = append(buildArgs, "-o", path.Join(h.OutFolder(), h.outTempFileName), h.MainFilePath())

	h.Cmd = exec.Command("go", buildArgs...)
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

	go io.Copy(h, stderr)
	go io.Copy(h, stdout)

	// Wait for build to complete
	if err := h.Cmd.Wait(); err != nil {
		return errors.Join(this, err)
	}

	// Only rename files if build was successful
	os.Remove(path.Join(h.OutFolder(), h.outFileName))

	err = os.Rename(
		path.Join(h.OutFolder(), h.outTempFileName),
		path.Join(h.OutFolder(), h.outFileName),
	)
	if err != nil {
		return errors.Join(this, err)
	}

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

func (h *GoCompiler) RunProgram() error {

	h.Cmd = exec.Command(h.outFileName, h.RunArguments()...)
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

	// Using context for cancellation
	done := make(chan struct{})

	go io.Copy(h, stderr)
	go io.Copy(h, stdout)

	// Monitor application state
	go func() {
		select {
		case <-h.ExitChan:
			h.Print("Received exit signal, stopping application...")
			h.StopProgram()
			close(done)
		case <-done:
			h.Print("Application closed")
		}
	}()

	// Wait for application completion
	go func() {
		err := h.Cmd.Wait()
		if err != nil {
			h.Print("Application closed with error:", err)
		} else {
			h.Print("Application closed successfully")
		}
		close(done)
	}()

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

	// BUILD
	err = h.BuildProgram()
	if err != nil {
		return errors.Join(this, err)
	}

	// RUN
	err = h.RunProgram()
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

// Write implementa io.Writer para capturar la salida
func (h *GoCompiler) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))

	if msg != "" {
		// Detectar automáticamente el tipo de mensaje
		msgType := detectMessageType(msg)

		// Si es un error
		if msgType == ErrorMsg {
			h.Print(errors.New(msg))
			return len(p), nil
		} else {
			// Si es un mensaje normal
			h.Print(msg)
			return len(p), nil
		}
	}

	return len(p), nil
}
