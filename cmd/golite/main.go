package main

import (
	"flag"
	"log"
	"os"

	"github.com/cdvelop/devtui" // ONLY import DevTUI in main.go
	"github.com/cdvelop/golite"
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
	logger := golite.NewLogger()

	// Create DevTUI instance
	ui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:  "GOLITE",
		ExitChan: exitChan,
		Color:    devtui.DefaultPalette(),
		Logger:   func(messages ...any) { logger.Logger(messages...) },
	})

	// Start GoLite - this will initialize handlers and start all goroutines
	// The Start function will block until exit
	golite.Start(rootDir, logger.Logger, ui, exitChan)
}
