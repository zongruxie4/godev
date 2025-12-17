package app

const serverTemplate = `package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "%s" // Template variable
	}

	publicDir := "public" // Template variable

	// Get current working directory for debugging
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting working directory: %%v", err)
	} else {
		log.Printf("Current working directory: %%s", wd)
	}

	// Check if public directory exists
	absPublicDir, err := filepath.Abs(publicDir)
	if err != nil {
		log.Printf("Error getting absolute path for public dir: %%v", err)
	} else {
		log.Printf("Public directory absolute path: %%s", absPublicDir)
	}

	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		log.Printf("WARNING: Public directory '%%s' does not exist!", publicDir)
	} else {
		log.Printf("Public directory '%%s' exists", publicDir)
	}

	// Serve static files
	fs := http.FileServer(http.Dir(publicDir))

	// Use a dedicated ServeMux so we can pass it to an http.Server
	mux := http.NewServeMux()
	mux.Handle("/", fs)

	// Health check endpoint
	mux.HandleFunc("/h", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "%s")
	})

	// Create http.Server with Addr and Handler set
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	fmt.Printf("Server port %%s — Servin static files from: %%s\n", port, publicDir)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
`
