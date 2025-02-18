package godev

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"sync"
)

type ServerHandler struct {
	*ServerConfig
	mainFileExternalServer string // eg: main.server.go
	internalServerRun      bool
	server                 *http.Server
	goCompiler             *GoCompiler
	goRun                  *GoRun
}

type ServerConfig struct {
	RootFolder                  string          // eg: web
	MainFileWithoutExtension    string          // eg: main.server
	ArgumentsForCompilingServer func() []string // eg: []string{"-X 'main.version=v1.0.0'"}
	ArgumentsToRunServer        func() []string // eg: []string{"dev" }
	PublicFolder                string          // eg: public
	AppPort                     string          // eg : 8080
	Writer                      io.Writer       // For logging output
	ExitChan                    chan bool       // Canal global para señalizar el cierre
}

func NewServerHandler(c *ServerConfig) *ServerHandler {

	var exe_ext = ""
	if runtime.GOOS == "windows" {
		exe_ext = ".exe"
	}

	sh := &ServerHandler{
		ServerConfig:           c,
		mainFileExternalServer: c.MainFileWithoutExtension + ".go",
		internalServerRun:      false,
		server:                 nil,
	}

	sh.goCompiler = NewGoCompiler(&GoCompilerConfig{
		Command:            "go",
		MainFilePath:       path.Join(c.RootFolder, sh.mainFileExternalServer),
		OutName:            c.MainFileWithoutExtension,
		Extension:          exe_ext,
		CompilingArguments: c.ArgumentsForCompilingServer,
		OutFolder:          c.RootFolder,
		Writer:             c.Writer,
	})

	sh.goRun = NewGoRun(&GoRunConfig{
		ExecProgramPath: path.Join(c.RootFolder, sh.goCompiler.outFileName),
		RunArguments:    c.ArgumentsToRunServer,
		ExitChan:        c.ExitChan,
		Writer:          c.Writer,
	})

	return sh
}

// Start inicia el servidor como goroutine
func (h *ServerHandler) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Fprintln(h.Writer, "Server Start ...")

	if _, err := os.Stat(h.goCompiler.MainFilePath); os.IsNotExist(err) {
		// ejecutar el servidor interno de archivos estáticos
		h.StartInternalServerFiles()
	} else {
		// construir y ejecutar el servidor externo
		err := h.StartExternalServer()
		if err != nil {
			fmt.Fprintln(h.Writer, "starting external server:", err)
		}
	}
}

// event: create,write,remove,rename
func (h *ServerHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	if event == "write" {
		// Case 1: External server file was modified
		if fileName == h.mainFileExternalServer {
			fmt.Fprintln(h.Writer, "External server modified, restarting ...")

			// Stop internal server if running to avoid port conflicts
			if h.internalServerRun {
				if err := h.StopInternalServer(); err != nil {
					return fmt.Errorf("stopping internal server: %w", err)
				}
			}

			// Restart external server with new changes
			return h.RestartExternalServer()
		}

		// Case 2: Shared Go file was modified
		if !h.internalServerRun {
			fmt.Fprintln(h.Writer, "Shared Go file modified, restarting external server ...")
			return h.RestartExternalServer()
		}
	}

	// Case 3: External server file was created for first time
	if event == "create" && fileName == h.mainFileExternalServer {
		fmt.Fprintln(h.Writer, "New external server detected")

		// Stop internal server if running
		if h.internalServerRun {
			if err := h.StopInternalServer(); err != nil {
				return fmt.Errorf("stopping internal server: %w", err)
			}
		}

		// Start the new external server
		return h.StartExternalServer()
	}

	return nil
}

func (h *ServerHandler) StartInternalServerFiles() {
	// Crear el servidor de archivos estáticos

	publicFolder := path.Join(h.RootFolder, h.PublicFolder)

	fs := http.FileServer(http.Dir(publicFolder))

	// Configurar el servidor HTTP
	h.server = &http.Server{
		Addr:    ":" + h.AppPort,
		Handler: fs,
	}

	fmt.Fprintln(h.Writer, "Godev Server Files:", publicFolder, "Running port:", h.AppPort)
	// Iniciar el servidor en una goroutine
	h.internalServerRun = true

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(h.Writer, "Internal Server Files error:", err)
		}
	}()
}

func (h *ServerHandler) StartExternalServer() error {
	this := errors.New("StartExternalServer")

	// COMPILE
	// Check if executable exists
	if _, err := os.Stat(h.goRun.ExecProgramPath); os.IsNotExist(err) {
		// COMPILE only if executable doesn't exist
		err := h.goCompiler.CompileProgram()
		if err != nil {
			return errors.Join(this, err)
		}
	}

	// RUN
	err := h.goRun.RunProgram()
	if err != nil {
		return errors.Join(this, err)
	}

	return nil
}

func (h *ServerHandler) StopInternalServer() error {
	if h.server != nil {
		h.internalServerRun = false
		// fmt.Fprintln(h.Writer,"Internal Server Stop")
		return h.server.Close()
	}
	return nil
}

func (h *ServerHandler) RestartInternalServer() error {
	err := h.StopInternalServer()
	if err != nil {
		return err
	}

	h.StartInternalServerFiles()
	return nil
}

func (h *ServerHandler) RestartExternalServer() error {
	var this = errors.New("Restart External Server")

	// STOP
	err := h.goRun.StopProgram()
	if err != nil {
		return errors.Join(this, errors.New("StopProgram"), err)

	}

	// COMPILE
	err = h.goCompiler.CompileProgram()
	if err != nil {
		return errors.Join(this, errors.New("CompileProgram"), err)
	}

	// RUN
	err = h.goRun.RunProgram()
	if err != nil {
		return errors.Join(this, errors.New("RunProgram"), err)
	}

	return nil
}

func (h *ServerHandler) UnobservedFiles() []string {
	return h.goCompiler.UnobservedFiles()
}
