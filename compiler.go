package godev

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/svg"
)

type Compiler struct {
	*CompilerConfig
	cssHandler *fileHandler
	jsHandler  *fileHandler
	min        *minify.M
}

type CompilerConfig struct {
	BuildDirectory         func() string         // eg: .web/static
	Println                func(messages ...any) // eg: fmt.Println
	WasmProjectTinyGoJsUse func() (bool, bool)   // eg: func() (bool,bool) { return true,true } = wasmProjectTinyGoJsUse()
}

type fileHandler struct {
	fileOutputName string                 // eg: main.js,style.css
	startCode      func() (string, error) // eg: "console.log('hello world')"
	files          []*File
	mediatype      string // eg: "text/html"
	buf            bytes.Buffer
}

type File struct {
	path    string
	content []byte
}

func New(config *CompilerConfig) *Compiler {
	c := &Compiler{
		CompilerConfig: config,
		cssHandler: &fileHandler{
			fileOutputName: "style.css",
			files:          []*File{},
			mediatype:      "text/css",
			buf:            bytes.Buffer{},
		},
		jsHandler: &fileHandler{
			fileOutputName: "main.js",
			files: []*File{
				{
					path:    "strict",
					content: []byte("'use strict';\n"),
				},
			},
			mediatype: "text/javascript",
			buf:       bytes.Buffer{},
		},
		min: minify.New(),
	}

	c.min.AddFunc("text/html", html.Minify)
	c.min.AddFunc("text/css", css.Minify)
	c.min.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)
	c.min.AddFunc("image/svg+xml", svg.Minify)

	c.jsHandler.startCode = c.StartCodeJS

	return c
}

func (c *Compiler) UpdateFileContentInMemory(filePath, extension string, content []byte) (*fileHandler, error) {
	file := &File{
		path:    filePath,
		content: content,
	}

	switch extension {
	case ".css":
		if idx := c.findFileIndex(c.cssHandler.files, filePath); idx != -1 {
			c.cssHandler.files[idx] = file
		} else {
			c.cssHandler.files = append(c.cssHandler.files, file)
		}
		return c.cssHandler, nil

	case ".js":
		if idx := c.findFileIndex(c.jsHandler.files, filePath); idx != -1 {
			c.jsHandler.files[idx] = file
		} else {
			c.jsHandler.files = append(c.jsHandler.files, file)
		}
		return c.jsHandler, nil
	}

	return nil, errors.New("UpdateFileContentInMemory extension: " + extension + " not found " + filePath)
}

func (c *Compiler) findFileIndex(files []*File, filePath string) int {
	for i, f := range files {
		if f.path == filePath {
			return i
		}
	}
	return -1
}

func (c *Compiler) UpdateFileOnDisk(filePath, extension string) error {
	var e = "UpdateFileOnDisk " + extension + " "
	if filePath == "" {
		return nil
	}

	c.Println("Compiling", extension, "...", filePath)

	time.Sleep(10 * time.Millisecond) // Esperar antes de intentar leer el archivo de nuevo

	//1- read file content from filePath
	content, err := os.ReadFile(filePath)
	if err != nil {
		return errors.New(e + err.Error())
	}

	fh, err := c.UpdateFileContentInMemory(filePath, extension, content)
	if err != nil {
		return errors.New(e + err.Error())
	}
	//fh.buf.Reset() // No es necesario resetear el buffer

	// if fh.startCode != nil {
	// 	startCode, err := fh.startCode()
	// 	if err != nil {
	// 		return errors.New(e + err.Error())
	// 	}
	// 	fh.buf.WriteString(startCode)
	// }

	// for _, f := range fh.files {
	// 	fh.buf.Write(f.content)
	// }

	var buf bytes.Buffer

	if extension == ".js" {
		startCode, err := c.StartCodeJS()
		if err != nil {
			return errors.New(e + err.Error())
		}
		buf.WriteString(startCode)
	}

	buf.Write(content)

	var minifiedBuf bytes.Buffer

	if err := c.min.Minify(fh.mediatype, &minifiedBuf, &buf); err != nil {
		return errors.New(e + err.Error())
	}

	if err := FileWrite(path.Join(c.BuildDirectory(), fh.fileOutputName), minifiedBuf); err != nil {
		return errors.New(e + err.Error())
	}

	return nil
}

func (c *Compiler) StartCodeJS() (out string, err error) {
	out = "'use strict';\n"

	// load wasm js code
	wasmType, TinyGoCompiler := c.WasmProjectTinyGoJsUse()
	if !wasmType {
		return out, nil
	}
	var wasmExecJsPath string
	if TinyGoCompiler {
		wasmExecJsPath, err = c.getWasmExecJsPathTinyGo()
	} else {
		wasmExecJsPath, err = c.getWasmExecJsPathGo()
	}
	if err != nil {
		return out, err
	}

	//  read wasm js code
	wasmJs, err := os.ReadFile(wasmExecJsPath)
	if err != nil {
		return out, errors.New("Error al leer el archivo wasm_exec.js: " + err.Error())
	}

	out += string(wasmJs)

	return out, nil
}

func (c *Compiler) getWasmExecJsPathTinyGo() (string, error) {

	path, err := exec.LookPath("tinygo")
	if err != nil {
		return "", errors.New("TinyGo no encontrado en el PATH. " + err.Error())
	}
	// Obtener el directorio de instalación
	tinyGoDir := filepath.Dir(path)

	// Limpiar la ruta y quitar "\bin"
	tinyGoDir = strings.TrimSuffix(tinyGoDir, "\\bin")

	// Construir la ruta completa al archivo wasm_exec.js
	return filepath.Join(tinyGoDir, "targets", "wasm_exec.js"), nil
}

func (c *Compiler) getWasmExecJsPathGo() (string, error) {

	// Obtener la ruta del directorio de instalación de Go desde la variable de entorno GOROOT
	path, er := exec.LookPath("go")
	if er != nil {
		return "", errors.New("Go no encontrado en el PATH. " + er.Error())
	}

	// Obtener el directorio de instalación
	GoDir := filepath.Dir(path)

	// Limpiar la ruta y quitar "\bin"
	GoDir = strings.TrimSuffix(GoDir, "\\bin")

	// Construir la ruta completa al archivo wasm_exec.js
	return filepath.Join(GoDir, "misc", "wasm", "wasm_exec.js"), nil
}
