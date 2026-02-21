package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/tinywasm/app"
	"github.com/tinywasm/deploy"
	"github.com/tinywasm/devbrowser"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/devtui"
	"github.com/tinywasm/kvdb"
	"github.com/tinywasm/server"
)

var Version = "dev"

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

	default:
		// Default Server implementation
		srvFactory = func() app.ServerInterface {
			return server.New().
				SetLogger(logger.Logger).
				SetExitChan(exitChan).
				SetStore(db).
				SetUI(ui).
				SetOpenBrowser(browser.OpenBrowser).
				SetGitIgnoreAdd(gitHandler.GitIgnoreAdd)
		}
	}

	keys := deploy.NewSystemKeyManager()

	// Start TinyWasm
	app.Start(startDir, logger.Logger, ui, browser, db, exitChan, srvFactory, githubAuth, gitHandler, goModHandler, keys, ui)
}
