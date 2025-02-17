package godev

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
)

type GoCompilerConfig struct {
	Command            string          // eg: "go", "tinygo"
	MainFilePath       string          // eg: web/main.server.go, web/main.wasm.go
	OutName            string          // eg: app, user, main.server
	Extension          string          // eg: .exe, .wasm
	CompilingArguments func() []string // eg: []string{"-X 'main.version=v1.0.0'"}
	OutFolder          string          // eg: web, web/public/wasm
	Writer             io.Writer
}

type GoCompiler struct {
	*GoCompilerConfig
	Cmd             *exec.Cmd
	outFileName     string // eg: main.exe, app
	outTempFileName string /// eg: app_temp.exe
}

func NewGoCompiler(c *GoCompilerConfig) *GoCompiler {

	return &GoCompiler{
		GoCompilerConfig: c,
		Cmd:              &exec.Cmd{},
		outFileName:      c.OutName + c.Extension,
		outTempFileName:  c.OutName + "_temp" + c.Extension,
	}
}

// eg: main.exe, main_temp.exe
func (h *GoCompiler) UnobservedFiles() []string {
	return []string{
		h.outFileName,
		h.outTempFileName,
	}
}

func (h *GoCompiler) CompileProgram() error {
	var this = errors.New("CompileProgram")
	buildArgs := []string{"build"}
	ldFlags := []string{}

	if h.CompilingArguments != nil {
		args := h.CompilingArguments()
		for _, arg := range args {
			if strings.HasPrefix(arg, "-X") { // eg: -X 'main.version=v1.0.0'
				ldFlags = append(ldFlags, arg)
			} else {
				buildArgs = append(buildArgs, arg)
			}
		}
	}

	// Add ldflags if any were found
	if len(ldFlags) > 0 {
		buildArgs = append(buildArgs, "-ldflags="+strings.Join(ldFlags, " "))
	}

	buildArgs = append(buildArgs, "-o", path.Join(h.OutFolder, h.outTempFileName), h.MainFilePath)
	h.Cmd = exec.Command(h.Command, buildArgs...)

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

	if err := h.Cmd.Wait(); err != nil {
		return errors.Join(this, err)
	}

	// rename temp file to final file name
	err = os.Rename(
		path.Join(h.OutFolder, h.outTempFileName),
		path.Join(h.OutFolder, h.outFileName),
	)
	if err != nil {
		return errors.Join(this, err)
	}

	return nil
}
