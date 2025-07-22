package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cdvelop/devtui"
	"github.com/cdvelop/godev"
)

// ServerHandler - Controlador dinámico que se actualiza
type ServerHandler struct {
	port     string
	status   string
	lastOpID string
}

func (h *ServerHandler) Name() string                 { return "ServerHandler" }
func (h *ServerHandler) SetLastOperationID(id string) { h.lastOpID = id }
func (h *ServerHandler) GetLastOperationID() string   { return h.lastOpID }
func (h *ServerHandler) Label() string                { return "Port" }
func (h *ServerHandler) Value() string                { return h.port }
func (h *ServerHandler) Editable() bool               { return true }
func (h *ServerHandler) Timeout() time.Duration       { return 3 * time.Second }
func (h *ServerHandler) Change(newValue any) (string, error) {
	port := strings.TrimSpace(newValue.(string))
	if port == "" {
		return "", fmt.Errorf("port cannot be empty")
	}

	// Simular configuración del servidor
	h.status = "configuring..."
	time.Sleep(1 * time.Second)

	h.port = port
	h.status = "running"
	return fmt.Sprintf("Server started on port %s", port), nil
}

// StatusHandler - Muestra el estado actual del servidor
type StatusHandler struct {
	server   *ServerHandler
	lastOpID string
}

func (h *StatusHandler) Name() string                 { return "StatusHandler" }
func (h *StatusHandler) SetLastOperationID(id string) { h.lastOpID = id }
func (h *StatusHandler) GetLastOperationID() string   { return h.lastOpID }
func (h *StatusHandler) Label() string                { return "Status" }
func (h *StatusHandler) Value() string                { return h.server.status }
func (h *StatusHandler) Editable() bool               { return false }
func (h *StatusHandler) Timeout() time.Duration       { return 2 * time.Second }
func (h *StatusHandler) Change(newValue any) (string, error) {
	// Simular verificación de estado
	time.Sleep(500 * time.Millisecond)

	if h.server.status == "running" {
		return fmt.Sprintf("Server healthy on port %s - Uptime: %s",
			h.server.port, time.Now().Format("15:04:05")), nil
	}
	return "Server status check completed", nil
}

// BuildHandler - Acción de construcción del proyecto
type BuildHandler struct {
	rootDir  string
	lastOpID string
}

func (h *BuildHandler) Name() string                 { return "BuildHandler" }
func (h *BuildHandler) SetLastOperationID(id string) { h.lastOpID = id }
func (h *BuildHandler) GetLastOperationID() string   { return h.lastOpID }
func (h *BuildHandler) Label() string                { return "Build Project" }
func (h *BuildHandler) Value() string                { return "Press Enter to build" }
func (h *BuildHandler) Editable() bool               { return false }
func (h *BuildHandler) Timeout() time.Duration       { return 10 * time.Second }
func (h *BuildHandler) Change(newValue any) (string, error) {
	// Simular proceso de construcción
	time.Sleep(2 * time.Second)
	return fmt.Sprintf("Build completed successfully in %s", h.rootDir), nil
}

// ModeHandler - Selector de modo de desarrollo
type ModeHandler struct {
	mode     string
	lastOpID string
}

func (h *ModeHandler) Name() string                 { return "ModeHandler" }
func (h *ModeHandler) SetLastOperationID(id string) { h.lastOpID = id }
func (h *ModeHandler) GetLastOperationID() string   { return h.lastOpID }
func (h *ModeHandler) Label() string                { return "Dev Mode" }
func (h *ModeHandler) Value() string                { return h.mode }
func (h *ModeHandler) Editable() bool               { return true }
func (h *ModeHandler) Timeout() time.Duration       { return 1 * time.Second }
func (h *ModeHandler) Change(newValue any) (string, error) {
	mode := strings.TrimSpace(newValue.(string))
	validModes := []string{"dev", "debug", "release"}

	for _, valid := range validModes {
		if mode == valid {
			h.mode = mode
			return fmt.Sprintf("Development mode changed to: %s", mode), nil
		}
	}

	return "", fmt.Errorf("invalid mode. Use: %s", strings.Join(validModes, ", "))
}

func main() {
	// Initialize root directory
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting current working directory:", err)
		return
	}

	// Create a Logger instance
	logger := godev.NewLogger()

	// Crear instancia TUI con configuración dinámica
	tui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:   "GoDEV",
		ExitChan:  make(chan bool),
		LogToFile: logger.LogToFile,
	})

	// Crear handlers interconectados
	serverHandler := &ServerHandler{port: "8080", status: "stopped"}
	statusHandler := &StatusHandler{server: serverHandler}

	// Tab 1: Configuración del Servidor (con controlador dinámico)
	tui.NewTabSection("Server", "Live server configuration").
		NewField(serverHandler).
		NewField(statusHandler)

	// Tab 2: Desarrollo y Construcción
	tui.NewTabSection("Development", "Project build and settings").
		NewField(&BuildHandler{rootDir: rootDir}).
		NewField(&ModeHandler{mode: "dev"})

	// Iniciar simulación de actualizaciones en segundo plano
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if serverHandler.status == "running" {
					// El StatusHandler se actualizará automáticamente cuando se ejecute
				}
			}
		}
	}()

	// Iniciar TUI
	var wg sync.WaitGroup
	wg.Add(1)
	go tui.Start(&wg)
	wg.Wait()
}
