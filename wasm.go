package godev

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

type WasmCompiler struct {
	*WasmConfig
}

type WasmConfig struct {
	BuildDirectory func() string         // eg: web/public/wasm
	Print          func(messages ...any) // eg: fmt.Println
}

func NewWasmCompiler(wc *WasmConfig) *WasmCompiler {
	return &WasmCompiler{
		WasmConfig: wc,
	}

}

func (w *WasmCompiler) WasmProjectTinyGoJsUse() (bool, bool) {

	return false, false
}

func getWasmExecJsPathTinyGo() (string, error) {
	path, err := exec.LookPath("tinygo")
	if err != nil {
		return "", errors.New("TinyGo not found in PATH. " + err.Error())
	}
	// Get installation directory
	tinyGoDir := filepath.Dir(path)
	// Clean path and remove "\bin"
	tinyGoDir = strings.TrimSuffix(tinyGoDir, "\\bin")
	// Build complete path to wasm_exec.js file
	return filepath.Join(tinyGoDir, "targets", "wasm_exec.js"), nil
}

func getWasmExecJsPathGo() (string, error) {
	// Get Go installation directory path from GOROOT environment variable
	path, er := exec.LookPath("go")
	if er != nil {
		return "", errors.New("Go not found in PATH. " + er.Error())
	}
	// Get installation directory
	GoDir := filepath.Dir(path)
	// Clean path and remove "\bin"
	GoDir = strings.TrimSuffix(GoDir, "\\bin")
	// Build complete path to wasm_exec.js file
	return filepath.Join(GoDir, "misc", "wasm", "wasm_exec.js"), nil
}
