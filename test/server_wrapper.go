package test

import (
	"net/http"

	"github.com/tinywasm/server"
)

// TestServerWrapper wraps server.ServerHandler to satisfy app.ServerInterface
// and allow void-return setters for type assertion in app package.
type TestServerWrapper struct {
	*server.ServerHandler
}

// RegisterRoutes adapts the signature to match ServerInterface (void return)
func (w *TestServerWrapper) RegisterRoutes(fn func(*http.ServeMux)) {
	w.ServerHandler.RegisterRoutes(fn)
}

// Setters with void return for configuration via type assertion
func (w *TestServerWrapper) SetAppRootDir(dir string)                 { w.ServerHandler.SetAppRootDir(dir) }
func (w *TestServerWrapper) SetSourceDir(dir string)                  { w.ServerHandler.SetSourceDir(dir) }
func (w *TestServerWrapper) SetOutputDir(dir string)                  { w.ServerHandler.SetOutputDir(dir) }
func (w *TestServerWrapper) SetPublicDir(dir string)                  { w.ServerHandler.SetPublicDir(dir) }
func (w *TestServerWrapper) SetMainInputFile(name string)             { w.ServerHandler.SetMainInputFile(name) }
func (w *TestServerWrapper) SetPort(port string)                      { w.ServerHandler.SetPort(port) }
func (w *TestServerWrapper) SetHTTPS(enabled bool)                    { w.ServerHandler.SetHTTPS(enabled) }
func (w *TestServerWrapper) SetDisableGlobalCleanup(disable bool)     { w.ServerHandler.SetDisableGlobalCleanup(disable) }
func (w *TestServerWrapper) SetCompileArgs(fn func() []string)        { w.ServerHandler.SetCompileArgs(fn) }
func (w *TestServerWrapper) SetRunArgs(fn func() []string)            { w.ServerHandler.SetRunArgs(fn) }
func (w *TestServerWrapper) SetOnExternalModeExecution(fn func(bool)) { w.ServerHandler.SetOnExternalModeExecution(fn) }
func (w *TestServerWrapper) SetExternalServerMode(external bool) error {
	return w.ServerHandler.SetExternalServerMode(external)
}
