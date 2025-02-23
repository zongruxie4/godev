package godev

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type WasmHandler struct {
	*WasmConfig
	ModulesFolder  string // default "modules". for test change eg: "test/modules"
	mainInputFile  string // eg: main.wasm.go
	mainOutputFile string // eg: main.wasm

	goWasmJsCache     string
	tinyGoWasmJsCache string
}

type WasmConfig struct {
	// root web folder and subfolder eg: "web","public"
	WebFilesFolder func() (string, string)
	Writer         io.Writer // For logging output
}

func NewWasmCompiler(c *WasmConfig) *WasmHandler {

	w := &WasmHandler{
		WasmConfig:     c,
		ModulesFolder:  "modules",
		mainInputFile:  "main.wasm.go",
		mainOutputFile: "main.wasm",
	}

	return w
}

// event: create, remove, write, rename
func (h *WasmHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	const e = "NewFileEvent Wasm"

	if filePath == "" {
		return errors.New(e + "filePath is empty")
	}

	fmt.Fprint(h.Writer, "Wasm", extension, event, "...", filePath)

	var inputFilePath, outputFilePath string

	// verificar si el archivo  es un modulo wasm independiente
	if fileName != h.mainInputFile {

		moduleName, err := GetModuleName(filePath)
		if err != nil {
			return errors.New(e + "GetModuleName: " + err.Error())
		}
		fmt.Fprint(h.Writer, "Module Name: "+moduleName)

		//create main module wasm path eg: modules/users/wasm/users.wasm.go
		inputFilePath = path.Join(h.ModulesFolder, moduleName, "wasm", moduleName+".wasm.go")
		outputFilePath = path.Join(h.wasmFilesOutputDirectory(), moduleName+".wasm")

	} else {
		// el fichero es wasm principal main.wasm.go
		inputFilePath = path.Join(h.wasmFilesOutputDirectory(), h.mainInputFile)
		outputFilePath = h.OutputPathMainFileWasm()
	}

	// Check if the ...wasm/moduleName.wasm.go file exists
	if _, err := os.Stat(inputFilePath); err != nil {
		// webAssembly not found
		return nil
	}

	if event != "write" {
		return nil
	}

	var cmd *exec.Cmd
	var flags string

	// log.Println("*** mainInputFile: ", mainInputFile)
	// Adjust compilation parameters according to configuration
	if h.TinyGoCompiler() {
		// fmt.Println("*** WASM TINYGO COMPILATION ***")
		cmd = exec.Command("tinygo", "build", "-o", outputFilePath, "-target", "wasm", "--no-debug", "-ldflags", flags, inputFilePath)

	} else {
		// normal compilation...

		cmd = exec.Command("go", "build", "-o", outputFilePath, "-tags", "dev", "-ldflags", flags, "-v", inputFilePath)
		// cmd = exec.Command("go", "build", "-o", outputFilePath, "-tags", "dev", "-ldflags", "-s -w", "-v", inputFilePath)
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	}

	output, er := cmd.CombinedOutput()
	if er != nil {
		return errors.New("compiling to WebAssembly error: " + er.Error() + " string(output):" + string(output))
	}

	// Check if the wasm file was created correctly
	if _, er := os.Stat(outputFilePath); er != nil {
		return errors.New("the WebAssembly file was not created correctly: " + er.Error())
	}

	// fmt.Printf("WebAssembly compiled successfully and saved in %s\n", outputFilePath)

	return nil
}

// ej: web/public/wasm/main.wasm
func (w *WasmHandler) OutputPathMainFileWasm() string {
	return path.Join(w.wasmFilesOutputDirectory(), w.mainOutputFile)
}

// eg: web/public/wasm
func (w *WasmHandler) wasmFilesOutputDirectory() string {
	rootFolder, subfolder := w.WebFilesFolder()
	return path.Join(rootFolder, subfolder, "wasm")
}

// eg: main.wasm
func (w *WasmHandler) UnobservedFiles() []string {
	return []string{
		w.mainOutputFile,
		// add wasm name modules here
	}
}

func (w *WasmHandler) WasmProjectTinyGoJsUse() (bool, bool) {

	return false, false
}

func (h *WasmHandler) JavascriptForInitializing() (js string, err error) {

	// load wasm js code
	wasmType, TinyGoCompiler := h.WasmProjectTinyGoJsUse()
	if !wasmType {
		return
	}

	// Return appropriate cached content if available
	if TinyGoCompiler && h.tinyGoWasmJsCache != "" {
		return h.tinyGoWasmJsCache, nil
	} else if !TinyGoCompiler && h.goWasmJsCache != "" {
		return h.goWasmJsCache, nil
	}

	var wasmExecJsPath string
	if TinyGoCompiler {
		wasmExecJsPath, err = h.getWasmExecJsPathTinyGo()
	} else {
		wasmExecJsPath, err = h.getWasmExecJsPathGo()
	}
	if err != nil {
		return "", err
	}

	//  read wasm js code
	wasmJs, err := os.ReadFile(wasmExecJsPath)
	if err != nil {
		return "", errors.New("reading wasm_exec.js file: " + err.Error())
	}

	stringWasmJs := string(wasmJs)

	// Store in appropriate cache
	if TinyGoCompiler {
		h.tinyGoWasmJsCache = stringWasmJs
	} else {
		h.goWasmJsCache = stringWasmJs
	}

	return stringWasmJs, nil
}

// returns if TinyGo compiler should be used
func (w *WasmHandler) TinyGoCompiler() bool {

	return false
}

func (w *WasmHandler) getWasmExecJsPathTinyGo() (string, error) {
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

func (w *WasmHandler) getWasmExecJsPathGo() (string, error) {
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
