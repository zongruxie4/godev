package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/tinywasm/app"
	"github.com/tinywasm/devbrowser"
	"github.com/tinywasm/devflow"
	"github.com/tinywasm/devtui" // ONLY import DevTUI in main.go
	"github.com/tinywasm/kvdb"
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
	// logger.SetRootDir initialized later after finding project root

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
	// Cache disabled by default, but we can explicitly set it or use flags if needed
	browser := devbrowser.New(ui, db, exitChan)
	browser.SetLog(func(messages ...any) { logger.Logger(messages...) })

	// Create GitHub Auth handler for TUI integration
	githubAuth := devflow.NewGitHubAuth()

	// Start TinyWasm - this will initialize handlers and start all goroutines
	// The Start function will block until exit
	// Pass ui and browser as interfaces
	app.Start(startDir, logger.Logger, ui, browser, db, exitChan, githubAuth, gitHandler, goModHandler, ui)
}
