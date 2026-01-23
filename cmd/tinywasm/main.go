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

func main() {
	flag.Parse()

	// Initialize root directory
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting current working directory:", err)
		return
	}

	exitChan := make(chan bool)

	// Create a Logger instance
	logger := app.NewLogger()
	logger.SetRootDir(rootDir) // Initialize logger to write to logs.log

	// Create DevTUI instance
	ui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:  "TINYWASM",
		ExitChan: exitChan,
		Color:    devtui.DefaultPalette(),
		Logger:   func(messages ...any) { logger.Logger(messages...) },
	})

	// Initialize DB
	db, err := kvdb.New(filepath.Join(rootDir, ".env"), logger.Logger, &app.FileStore{})
	if err != nil {
		logger.Logger("Failed to initialize database:", err)
		return
	}

	// Create DevBrowser instance
	browser := devbrowser.New(ui, db, exitChan)
	browser.SetLog(func(messages ...any) { logger.Logger(messages...) })

	// Create GitHub Auth handler for TUI integration
	githubAuth := devflow.NewGitHubAuth()

	// Start TinyWasm - this will initialize handlers and start all goroutines
	// The Start function will block until exit
	// Pass ui and browser as interfaces
	app.Start(rootDir, logger.Logger, ui, browser, db, exitChan, githubAuth, ui)
}
