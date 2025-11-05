package main

import (
	"log"
	"os"

	"github.com/cdvelop/devtui" // ONLY import DevTUI in main.go
	"github.com/cdvelop/golite"
)

func main() {
	// Initialize root directory
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting current working directory:", err)
		return
	}

	exitChan := make(chan bool)

	// Create a Logger instance
	logger := golite.NewLogger()

	// Create DevTUI instance (ONLY in main.go)
	ui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:  "GOLITE",
		ExitChan: exitChan,
		Color:    devtui.DefaultPalette(),
		Logger:   func(messages ...any) { logger.Logger(messages...) },
	})

	// Pass UI as interface to Start - GOLITE doesn't know it's DevTUI
	// Start runs in background
	go golite.Start(rootDir, logger.Logger, ui, exitChan)

	// Give Start time to initialize ActiveHandler and components
	// (a small delay ensures h.config and other dependencies are ready)
	// In production this is fine, tests don't call main.go
	for golite.ActiveHandler == nil {
		// Wait for ActiveHandler to be set by Start()
	}

	// Start MCP server for LLM integration after handler is initialized
	// This blocks on stdio, waiting for MCP client connections
	golite.ActiveHandler.ServeMCP()
}
