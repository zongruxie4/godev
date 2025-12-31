package main

import (
	"flag"
	"log"
	"os"

	"github.com/tinywasm/app"
	"github.com/tinywasm/devtui" // ONLY import DevTUI in main.go
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

	// Create DevTUI instance
	ui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:  "TINYWASM",
		ExitChan: exitChan,
		Color:    devtui.DefaultPalette(),
		Logger:   func(messages ...any) { logger.Logger(messages...) },
	})

	// Start TinyWasm - this will initialize handlers and start all goroutines
	// The Start function will block until exit
	// Pass ui as MCP tool handler so devtui_get_section_logs tool is registered
	app.Start(rootDir, logger.Logger, ui, exitChan, ui)
}
