package main

import (
	"flag"
	"log"
	"os"

	"github.com/tinywasm/devtui" // ONLY import DevTUI in main.go
	"github.com/tinywasm/app"
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
	logger := tinywasm.NewLogger()

	// Create DevTUI instance
	ui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:  "TINYWASM",
		ExitChan: exitChan,
		Color:    devtui.DefaultPalette(),
		Logger:   func(messages ...any) { logger.Logger(messages...) },
	})

	// Start TinyWasm - this will initialize handlers and start all goroutines
	// The Start function will block until exit
	tinywasm.Start(rootDir, logger.Logger, ui, exitChan)
}
