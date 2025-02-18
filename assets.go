package godev

import (
	"bytes"
	"errors"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/svg"
)

type AssetsHandler struct {
	*AssetsConfig
	cssHandler *fileHandler
	jsHandler  *fileHandler
	min        *minify.M

	goWasmJsCache     string
	tinyGoWasmJsCache string
}

type AssetsConfig struct {
	ThemeFolder            func() string         // eg: web/theme
	WebFilesFolder         func() string         // eg: web/static, web/public, web/assets
	Print                  func(messages ...any) // eg: fmt.Println
	WasmProjectTinyGoJsUse func() (bool, bool)   // eg: func() (bool,bool) { return true,true } = wasmProjectTinyGoJsUse()
}

type fileHandler struct {
	fileOutputName string                 // eg: main.js,style.css
	startCode      func() (string, error) // eg: "console.log('hello world')"
	files          []*File
	mediatype      string // eg: "text/html", "text/css", "image/svg+xml"
}

type File struct {
	path    string
	content []byte
}

func NewAssetsCompiler(config *AssetsConfig) *AssetsHandler {
	c := &AssetsHandler{
		AssetsConfig: config,
		cssHandler: &fileHandler{
			fileOutputName: "style.css",
			files:          []*File{},
			mediatype:      "text/css",
		},
		jsHandler: &fileHandler{
			fileOutputName: "main.js",
			files:          []*File{},
			mediatype:      "text/javascript",
		},
		min: minify.New(),
	}

	c.min.AddFunc("text/html", html.Minify)
	c.min.AddFunc("text/css", css.Minify)
	// c.min.AddFunc("text/javascript", js.Minify)
	c.min.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)
	c.min.AddFunc("image/svg+xml", svg.Minify)

	c.jsHandler.startCode = c.StartCodeJS

	return c
}

func (c *AssetsHandler) UpdateFileContentInMemory(filePath, extension, event string, content []byte) (*fileHandler, error) {
	file := &File{
		path:    filePath,
		content: content,
	}

	switch extension {
	case ".css":
		if event == "remove" {
			if idx := c.findFileIndex(c.cssHandler.files, filePath); idx != -1 {
				c.cssHandler.files = append(c.cssHandler.files[:idx], c.cssHandler.files[idx+1:]...)
			}
		} else {
			if idx := c.findFileIndex(c.cssHandler.files, filePath); idx != -1 {
				c.cssHandler.files[idx] = file
			} else {
				c.cssHandler.files = append(c.cssHandler.files, file)
			}
		}
		return c.cssHandler, nil

	case ".js":
		if event == "remove" {
			if idx := c.findFileIndex(c.jsHandler.files, filePath); idx != -1 {
				c.jsHandler.files = append(c.jsHandler.files[:idx], c.jsHandler.files[idx+1:]...)
			}
		} else {
			if idx := c.findFileIndex(c.jsHandler.files, filePath); idx != -1 {
				c.jsHandler.files[idx] = file
			} else {
				c.jsHandler.files = append(c.jsHandler.files, file)
			}
		}
		return c.jsHandler, nil
	}

	return nil, errors.New("UpdateFileContentInMemory extension: " + extension + " not found " + filePath)
}

func (c *AssetsHandler) findFileIndex(files []*File, filePath string) int {
	for i, f := range files {
		if f.path == filePath {
			return i
		}
	}
	return -1
}

// event: create, remove, write, rename
func (c *AssetsHandler) NewFileEvent(fileName, extension, filePath, event string) error {
	var e = "NewFileEvent " + extension + " " + event
	if filePath == "" {
		return errors.New(e + "filePath is empty")
	}

	c.Print("Asset", event, extension, "...", filePath)

	time.Sleep(10 * time.Millisecond) // Esperar antes de intentar leer el archivo de nuevo

	//1- read file content from filePath
	content, err := os.ReadFile(filePath)
	if err != nil {
		return errors.New(e + err.Error())
	}

	fh, err := c.UpdateFileContentInMemory(filePath, extension, event, content)
	if err != nil {
		return errors.New(e + err.Error())
	}

	var buf bytes.Buffer

	if fh.startCode != nil {
		startCode, err := fh.startCode()
		if err != nil {
			return errors.New(e + err.Error())
		}
		buf.WriteString(startCode)
	}

	for _, f := range fh.files {
		buf.Write(f.content)
	}
	var minifiedBuf bytes.Buffer

	if err := c.min.Minify(fh.mediatype, &minifiedBuf, &buf); err != nil {
		return errors.New(e + err.Error())
	}

	if err := FileWrite(path.Join(c.WebFilesFolder(), fh.fileOutputName), minifiedBuf); err != nil {
		return errors.New(e + err.Error())
	}

	return nil
}

func (c *AssetsHandler) UnobservedFiles() []string {
	return []string{
		c.cssHandler.fileOutputName,
		c.jsHandler.fileOutputName,
	}
}

func (c *AssetsHandler) StartCodeJS() (out string, err error) {
	out = "'use strict';"

	// load wasm js code
	wasmType, TinyGoCompiler := c.WasmProjectTinyGoJsUse()
	if !wasmType {
		return out, nil
	}

	// Return appropriate cached content if available
	if TinyGoCompiler && c.tinyGoWasmJsCache != "" {
		return out + c.tinyGoWasmJsCache, nil
	} else if !TinyGoCompiler && c.goWasmJsCache != "" {
		return out + c.goWasmJsCache, nil
	}

	var wasmExecJsPath string
	if TinyGoCompiler {
		wasmExecJsPath, err = getWasmExecJsPathTinyGo()
	} else {
		wasmExecJsPath, err = getWasmExecJsPathGo()
	}
	if err != nil {
		return out, err
	}

	//  read wasm js code
	wasmJs, err := os.ReadFile(wasmExecJsPath)
	if err != nil {
		return out, errors.New("Error reading wasm_exec.js file: " + err.Error())
	}

	// Store in appropriate cache
	if TinyGoCompiler {
		c.tinyGoWasmJsCache = string(wasmJs)
	} else {
		c.goWasmJsCache = string(wasmJs)
	}

	return out + string(wasmJs), nil
}
