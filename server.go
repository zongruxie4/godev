package godev

import (
	"net/http"
	"os"
	"path"
	"sync"
)

type ServerHandler struct {
	*ServerConfig
	mainFilePath      string // eg: web/main.server.go
	internalServerRun bool
	server            *http.Server
}

type ServerConfig struct {
	RootFolder   string                // eg: web
	MainFile     string                // eg: main.server.go
	PublicFolder string                // eg: public
	AppPort      string                // eg : 8080
	Print        func(messages ...any) // eg: fmt.Println
	ExitChan     chan bool             // Canal global para señalizar el cierre
}

func NewServerHandler(c *ServerConfig) *ServerHandler {

	return &ServerHandler{
		ServerConfig: c,
		mainFilePath: path.Join(c.RootFolder, c.MainFile),
	}
}

// Start inicia el servidor como goroutine
func (h *ServerHandler) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	h.Print("Server Start")

	if _, err := os.Stat(h.mainFilePath); os.IsNotExist(err) {
		// ejecutar el servidor interno de archivos estáticos
		h.startInternalServerFiles()
	} else {
		// construir y ejecutar el servidor externo
	}

	// Esperar señal de cierre
	// <-h.ExitChan
}

func (h *ServerHandler) UpdateFileOnDisk(fileName, filePath string) error {

	h.Print("Go File IsBackend")
	if h.IsMainFile(fileName) {
		h.Print("Go File IsMainFile")
	} else {
		h.Print("Go File IsNotMainFile")
	}

	return nil
}

func (h *ServerHandler) startInternalServerFiles() {
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

func (h *ServerHandler) IsMainFile(fileName string) bool {

	if fileName == h.MainFile {
		// servidor fue modificado ejecutar

		// estoy ejecutando el servidor interno?
		if h.internalServerRun {
			// cerrar el servidor interno
			err := h.Stop()
			if err != nil {
				h.Print("IsMainFile Stop", err)
			}
			return true
		}

		// ejecutar el servidor externo

		return true
	}

	return false
}

func (h *ServerHandler) Stop() error {
	if h.server != nil {
		h.internalServerRun = false
		h.Print("Server Stop")
		return h.server.Close()
	}
	return nil
}

func (h *ServerHandler) Restart() error {
	err := h.Stop()
	if err != nil {
		return err
	}

	h.startInternalServerFiles()
	return nil
}
