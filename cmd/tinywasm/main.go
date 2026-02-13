package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tinywasm/app"
	"github.com/tinywasm/devbrowser"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/devtui"
	"github.com/tinywasm/kvdb"
	"github.com/tinywasm/server"
	"github.com/tinywasm/wasi"
)

var Version = "dev"

// ServerWrapper wraps server.ServerHandler to satisfy app.ServerInterface
// and allow void-return setters for type assertion in app package.
type ServerWrapper struct {
	*server.ServerHandler
}

// RegisterRoutes adapts the signature to match ServerInterface (void return)
func (w *ServerWrapper) RegisterRoutes(fn func(*http.ServeMux)) {
	w.ServerHandler.RegisterRoutes(fn)
}

// Setters with void return for configuration via type assertion
func (w *ServerWrapper) SetAppRootDir(dir string)                 { w.ServerHandler.SetAppRootDir(dir) }
func (w *ServerWrapper) SetSourceDir(dir string)                  { w.ServerHandler.SetSourceDir(dir) }
func (w *ServerWrapper) SetOutputDir(dir string)                  { w.ServerHandler.SetOutputDir(dir) }
func (w *ServerWrapper) SetPublicDir(dir string)                  { w.ServerHandler.SetPublicDir(dir) }
func (w *ServerWrapper) SetMainInputFile(name string)             { w.ServerHandler.SetMainInputFile(name) }
func (w *ServerWrapper) SetPort(port string)                      { w.ServerHandler.SetPort(port) }
func (w *ServerWrapper) SetHTTPS(enabled bool)                    { w.ServerHandler.SetHTTPS(enabled) }
func (w *ServerWrapper) SetDisableGlobalCleanup(disable bool)     { w.ServerHandler.SetDisableGlobalCleanup(disable) }
func (w *ServerWrapper) SetCompileArgs(fn func() []string)        { w.ServerHandler.SetCompileArgs(fn) }
func (w *ServerWrapper) SetRunArgs(fn func() []string)            { w.ServerHandler.SetRunArgs(fn) }
func (w *ServerWrapper) SetOnExternalModeExecution(fn func(bool)) { w.ServerHandler.SetOnExternalModeExecution(fn) }

func main() {
	// NEW: Parse debug flag for unfiltered logging
	debugFlag := flag.Bool("debug", false, "Enable debug mode for unfiltered logs")
	flag.Parse()

	// Initialize start directory
	startDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting current working directory:", err)
		return
	}

	exitChan := make(chan bool)

	// Create a Logger instance
	logger := app.NewLogger()

	// Initialize GoMod Handler
	goModHandler := devflow.NewGoModHandler()
	// Initialize Git Handler
	gitHandler, err := devflow.NewGit()
	if err != nil {
		logger.Logger("Error initializing Git handler:", err)
	}

	if projectRoot, err := devflow.FindProjectRoot(startDir); err == nil {
		gitHandler.SetRootDir(projectRoot)
		goModHandler.SetRootDir(projectRoot)
		logger.SetRootDir(projectRoot)
	} else {
		gitHandler.SetRootDir(startDir)
		goModHandler.SetRootDir(startDir)
		logger.SetRootDir(startDir)
	}

	goModHandler.SetLog(logger.Logger)

	// Create DevTUI instance
	ui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:    "TINYWASM",
		AppVersion: Version,
		ExitChan:   exitChan,
		Color:      devtui.DefaultPalette(),
		Logger:     func(messages ...any) { logger.Logger(messages...) },
		Debug:      *debugFlag,
	})

	// Initialize DB
	db, err := kvdb.New(filepath.Join(startDir, ".env"), logger.Logger, &app.FileStore{})
	if err != nil {
		logger.Logger("Failed to initialize database:", err)
		return
	}

	// Create DevBrowser instance
	browser := devbrowser.New(ui, db, exitChan)
	browser.SetLog(func(messages ...any) { logger.Logger(messages...) })

	// Create GitHub Auth handler for TUI integration
	githubAuth := devflow.NewGitHubAuth()

	// Configure Server Factory
	// We use DB to decide which server implementation to use
	serverType, err := db.Get("TINYWASM_SERVER")
	if err != nil {
		serverType = "server" // Default fallback
	}

	var srvFactory app.ServerFactory

	switch serverType {
	case "wasi":
		// WASI implementation
		// Note: Currently wasi package v0.0.1 is minimal.
		// We implement a basic factory that might panic or fail if used until wasi is fully implemented.
		srvFactory = func() app.ServerInterface {
			// Placeholder for WASI server
			// Since we can't wrap it meaningfully without methods, we might need a different approach if wasi is required.
			// For now, we log a fatal error as returning nil would cause a panic later.
			log.Fatal("FATAL: WASI server implementation is incomplete and cannot be used.")
			return nil
		}
	default:
		// Default Server implementation
		srvFactory = func() app.ServerInterface {
			s := server.New().
				SetLogger(logger.Logger).
				SetExitChan(exitChan).
				SetStore(db).
				SetUI(ui).
				SetOpenBrowser(browser.OpenBrowser).
				SetGitIgnoreAdd(gitHandler.GitIgnoreAdd)

			return &ServerWrapper{s}
		}
	}

	// Start TinyWasm
	app.Start(startDir, logger.Logger, ui, browser, db, exitChan, srvFactory, githubAuth, gitHandler, goModHandler, ui)
}
