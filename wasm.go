package godev

import (
	"errors"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type WasmCompiler struct {
	*WasmConfig
	wasmProjectType    bool
	tinyGoCompiler     bool
	mainWasmOutputFile string // eg: main.wasm
	mainGoInputFile    string // eg: main.wasm.go
}

type WasmConfig struct {
	// root folder and subfolder eg: "web","public"
	WebFilesFolder func() (string, string)
	Print          func(messages ...any) // eg: fmt.Println
}

func NewWasmCompiler(c *WasmConfig) *WasmCompiler {

	w := &WasmCompiler{
		WasmConfig:         c,
		mainWasmOutputFile: "main.wasm",
		mainGoInputFile:    "main.wasm.go",
	}

	return w
}

// ej: web/public/wasm/main.wasm
func (w *WasmCompiler) OutputPathMainFileWasm() string {
	return path.Join(w.wasmFilesOutputDirectory(), w.mainWasmOutputFile)
}

// eg: web/public/wasm
func (w *WasmCompiler) wasmFilesOutputDirectory() string {
	rootFolder, subfolder := w.WebFilesFolder()
	return path.Join(rootFolder, subfolder, "wasm")
}

// eg: main.wasm
func (w *WasmCompiler) UnchangeableOutputFileNames() []string {
	return []string{
		w.mainWasmOutputFile,
		// add wasm name modules here
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
