package godev

import (
	"errors"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
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
	RootFolder                  string                // eg: web
	MainFileWithoutExtension    string                // eg: main.server
	ArgumentsForCompilingServer func() []string       // eg: []string{"-X 'main.version=v1.0.0'"}
	ArgumentsToRunServer        func() []string       // eg: []string{"dev" }
	PublicFolder                string                // eg: public
	AppPort                     string                // eg : 8080
	Print                       func(messages ...any) // eg: fmt.Println
	ExitChan                    chan bool             // Canal global para señalizar el cierre
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
		Writer:             sh,
	})

	sh.goRun = NewGoRun(&GoRunConfig{
		ExecProgramPath: path.Join(c.RootFolder, sh.goCompiler.outFileName),
		RunArguments:    c.ArgumentsToRunServer,
		ExitChan:        c.ExitChan,
		Writer:          sh,
	})

	return sh
}

// Start inicia el servidor como goroutine
func (h *ServerHandler) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	h.Print("Server Start...")

	if _, err := os.Stat(h.goCompiler.MainFilePath); os.IsNotExist(err) {
		// ejecutar el servidor interno de archivos estáticos
		h.StartInternalServerFiles()
	} else {
		// construir y ejecutar el servidor externo
		err := h.StartExternalServer()
		if err != nil {
			h.Print("starting external server:", err)
		}
	}
}

func (h *ServerHandler) UpdateFileOnDisk(fileName, filePath string) error {

	this := errors.New("UpdateFileOnDisk")

	if fileName == h.mainFileExternalServer { // servidor externo fue modificado ejecutar
		// estoy ejecutando el servidor interno?
		if h.internalServerRun {
			err := h.StopInternalServer() // cerrar el servidor interno
			if err != nil {
				return errors.Join(this, err)
			}
		}

		err := h.StartExternalServer() // ejecutar el servidor externo
		if err != nil {
			return errors.Join(this, err)
		}

	} else { // archivo go compartido fue modificado

		if !h.internalServerRun { // si estoy ejecutando el servidor externo
			err := h.RestartExternalServer()
			if err != nil {
				return errors.Join(this, err)
			}
		}

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

	h.Print("Godev Server Files:", publicFolder, "Running port:", h.AppPort)
	// Iniciar el servidor en una goroutine
	h.internalServerRun = true

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.Print("Internal Server Files error:", err)
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
		h.Print("Internal Server Stop")
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
	var this = errors.New("Restart")
	h.Print("Restart ...")

	// STOP
	err := h.goRun.StopProgram()
	if err != nil {
		return errors.Join(this, err)

	}

	// COMPILE
	err = h.goCompiler.CompileProgram()
	if err != nil {
		return errors.Join(this, err)
	}

	// RUN
	err = h.goRun.RunProgram()
	if err != nil {
		return errors.Join(this, err)
	}

	return nil
}

func (h *ServerHandler) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		h.Print(msg)
	}
	return len(p), nil
}

func (h *ServerHandler) UnchangeableOutputFileNames() []string {
	return h.goCompiler.UnchangeableOutputFileNames()
}
