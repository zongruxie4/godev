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

// TinyGoHandler implementa devtui.FieldHandler para el compilador TinyGo
type TinyGoFieldHandler struct {
	wasmHandler *tinywasm.TinyWasm
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

// BrowserHandler implementa devtui.FieldHandler para el navegador
type BrowserFieldHandler struct {
	browser *Browser
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

// Verificar que los handlers implementan la interfaz FieldHandler
var _ devtui.FieldHandler = (*ServerFieldHandler)(nil)
var _ devtui.FieldHandler = (*TinyGoFieldHandler)(nil)
var _ devtui.FieldHandler = (*BrowserFieldHandler)(nil)
