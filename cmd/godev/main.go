package main

import (
	"log"
	"os"

	"github.com/cdvelop/godev"
)

func main() {
	// Initialize root directory
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting current working directory:", err)
		return
	}

	// Create a Logger instance
	logger := godev.NewLogger()

	godev.Start(rootDir, logger.LogToFile)

}
