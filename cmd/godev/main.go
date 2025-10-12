package main

import (
	"log"
	"os"

	"github.com/cdvelop/godev"
	"github.com/cdvelop/devtui" // ONLY import DevTUI in main.go
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
	logger := godev.NewLogger()

	// Create DevTUI instance (ONLY in main.go)
	ui := devtui.NewTUI(&devtui.TuiConfig{
		AppName:  "GODEV",
		ExitChan: exitChan,
		Color:    devtui.DefaultPalette(),
		Logger:   func(messages ...any) { logger.Logger(messages...) },
	})

	// Pass UI as interface to Start - GODEV doesn't know it's DevTUI
	godev.Start(rootDir, logger.Logger, ui, exitChan)
}