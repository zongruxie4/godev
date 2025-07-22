// Ejemplo de cómo se actualizarían los handlers en godev/handlers.go
package godev

import (
	"fmt"
	"time"

	"github.com/cdvelop/devtui"
	"github.com/cdvelop/tinywasm"
)

// ServerHandler implementa devtui.FieldHandler para el manejo del servidor
type ServerFieldHandler struct {
	*ServerHandler
	lastOpID    string
	needsUpdate bool // Para controlar si actualizar mensajes existentes
}

func (h *ServerFieldHandler) Label() string {
	if h.internalServerRun {
		return "Server (Internal)"
	}
	return "Server (External)"
}

func (h *ServerFieldHandler) Value() string {
	if h.internalServerRun {
		return "Press Enter to restart internal server"
	}
	return "Press Enter to restart external server"
}

func (h *ServerFieldHandler) Editable() bool {
	return false // Action button, not editable field
}

func (h *ServerFieldHandler) Timeout() time.Duration {
	return 30 * time.Second
}

func (h *ServerFieldHandler) Change(newValue any) (string, error) {
	return h.RestartServer()
}

// NEW: WritingHandler methods
func (h *ServerFieldHandler) Name() string { return "MainServer" }
func (h *ServerFieldHandler) SetLastOperationID(lastOpID string) {
	h.lastOpID = lastOpID
}
func (h *ServerFieldHandler) GetLastOperationID() string {
	if h.needsUpdate {
		return h.lastOpID // Update existing message during server operations
	}
	return "" // Create new message
}

// TinyGoHandler implementa devtui.FieldHandler para el compilador TinyGo
type TinyGoFieldHandler struct {
	wasmHandler *tinywasm.TinyWasm
	lastOpID    string
	isCompiling bool
}

func NewTinyGoFieldHandler(wasm *tinywasm.TinyWasm) *TinyGoFieldHandler {
	return &TinyGoFieldHandler{wasmHandler: wasm}
}

func (h *TinyGoFieldHandler) Label() string {
	return "TinyGo Compiler"
}

func (h *TinyGoFieldHandler) Value() string {
	if h.wasmHandler.TinyGoCompiler() {
		return "Enabled (Press Enter to disable)"
	}
	return "Disabled (Press Enter to enable)"
}

func (h *TinyGoFieldHandler) Editable() bool {
	return false // Action button to toggle
}

func (h *TinyGoFieldHandler) Timeout() time.Duration {
	return 5 * time.Second
}

func (h *TinyGoFieldHandler) Change(newValue any) (string, error) {
	// Toggle TinyGo compiler state
	newState := !h.wasmHandler.TinyGoCompiler()
	return h.wasmHandler.SetTinyGoCompiler(newState)
}

// NEW: WritingHandler methods
func (h *TinyGoFieldHandler) Name() string { return "TinyWasm" }
func (h *TinyGoFieldHandler) SetLastOperationID(lastOpID string) {
	h.lastOpID = lastOpID
}
func (h *TinyGoFieldHandler) GetLastOperationID() string {
	if h.isCompiling {
		return h.lastOpID // Update existing compilation messages
	}
	return "" // Create new message
}

// BrowserHandler implementa devtui.FieldHandler para el navegador
type BrowserFieldHandler struct {
	browser  *Browser
	lastOpID string
}

func NewBrowserFieldHandler(browser *Browser) *BrowserFieldHandler {
	return &BrowserFieldHandler{browser: browser}
}

func (h *BrowserFieldHandler) Label() string {
	return "Web Browser"
}

func (h *BrowserFieldHandler) Value() string {
	return "Press Enter to open/reload"
}

func (h *BrowserFieldHandler) Editable() bool {
	return false // Action button
}

func (h *BrowserFieldHandler) Timeout() time.Duration {
	return 10 * time.Second
}

func (h *BrowserFieldHandler) Change(newValue any) (string, error) {
	err := h.browser.Reload()
	if err != nil {
		return "", fmt.Errorf("browser reload failed: %w", err)
	}
	return "Browser reloaded successfully", nil
}

// NEW: WritingHandler methods
func (h *BrowserFieldHandler) Name() string { return "WebBrowser" }
func (h *BrowserFieldHandler) SetLastOperationID(lastOpID string) {
	h.lastOpID = lastOpID
}
func (h *BrowserFieldHandler) GetLastOperationID() string {
	return "" // Always create new messages for browser operations
}

// NEW: Handlers independientes que necesitan escribir pero no son campos

// WatcherWritingHandler - Para el file watcher que no es un field
type WatcherWritingHandler struct {
	lastOpID   string
	isWatching bool
}

func (h *WatcherWritingHandler) Name() string { return "FileWatcher" }
func (h *WatcherWritingHandler) SetLastOperationID(lastOpID string) {
	h.lastOpID = lastOpID
}
func (h *WatcherWritingHandler) GetLastOperationID() string {
	if h.isWatching {
		return h.lastOpID // Update file change notifications
	}
	return "" // Create new message for different files
}

// AssetWritingHandler - Para el asset processor
type AssetWritingHandler struct {
	lastOpID     string
	isProcessing bool
}

func (h *AssetWritingHandler) Name() string { return "AssetProcessor" }
func (h *AssetWritingHandler) SetLastOperationID(lastOpID string) {
	h.lastOpID = lastOpID
}
func (h *AssetWritingHandler) GetLastOperationID() string {
	if h.isProcessing {
		return h.lastOpID // Update asset processing status
	}
	return "" // Create new message for different assets
}

// Verificar que los handlers implementan la interfaz FieldHandler
var _ devtui.FieldHandler = (*ServerFieldHandler)(nil)
var _ devtui.FieldHandler = (*TinyGoFieldHandler)(nil)
var _ devtui.FieldHandler = (*BrowserFieldHandler)(nil)
