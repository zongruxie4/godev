package app

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tinywasm/devflow"
	"github.com/tinywasm/mcpserve"
	"github.com/tinywasm/sse"
)

// BootstrapConfig holds dependencies for bootstrapping the application
type BootstrapConfig struct {
	StartDir        string
	McpMode         bool
	Debug           bool
	Logger          func(messages ...any)
	DB              DB
	GitHandler      devflow.GitClient
	GoModHandler    devflow.GoModInterface
	ServerFactory   ServerFactory
	TuiFactory      func(exitChan chan bool) TuiInterface
	BrowserFactory  func(ui TuiInterface, exitChan chan bool) BrowserInterface
	GitHubAuth      any
	McpToolHandlers []mcpserve.ToolProvider
}

// Bootstrap is the main entry point for the application logic
func Bootstrap(cfg BootstrapConfig) {
	// 1. Check if we should run as Daemon (headless) or Client (TUI)

	if cfg.McpMode {
		// Force Daemon mode
		runDaemon(cfg)
		return
	}

	// Check if port 3030 is open (meaning occupied by another instance)
	if isPortOpen("3030") {
		// Port occupied -> Run as Client
		runClient(cfg)
	} else {
		// Port free -> Start Daemon in background, then run Client
		if err := startDaemonProcess(cfg.StartDir); err != nil {
			fmt.Printf("Failed to start daemon: %v\n", err)
			os.Exit(1)
		}

        // Wait for port
		timeout := time.After(5 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		ready := false
		for !ready {
			select {
			case <-timeout:
				fmt.Println("Timeout waiting for daemon to start")
				os.Exit(1)
			case <-ticker.C:
				if isPortOpen("3030") {
					ready = true
				}
			}
		}
		runClient(cfg)
	}
}

func runDaemon(cfg BootstrapConfig) {
	// Headless mode
	// Create HeadlessTUI
	headlessTui := NewHeadlessTUI(cfg.Logger)

	// Start SSE server on port 3031
	sseSrv := startSSEServer()

	// We need to loop to support restart
	for {
		exitChan := make(chan bool)

		// Call Start
		// We pass headless=true
		// We pass headlessTui as ui

		// Wrap logger to broadcast to SSE
		logger := func(messages ...any) {
			cfg.Logger(messages...)
			if sseSrv != nil {
				msg := fmt.Sprint(messages...)
				sseSrv.Publish([]byte(msg))
			}
		}

		restart := Start(
			cfg.StartDir,
			logger,
			headlessTui,
			cfg.BrowserFactory(headlessTui, exitChan), // Daemon needs browser control
			cfg.DB,
			exitChan,
			cfg.ServerFactory,
            cfg.GitHubAuth,
            cfg.GitHandler,
            cfg.GoModHandler,
            true, // headless
            cfg.McpToolHandlers...,
        )

        if !restart {
            break
        }

        cfg.Logger("Restarting daemon...")
    }
}

func runClient(cfg BootstrapConfig) {
	// Client mode
	// Use real TUI
	exitChan := make(chan bool)
	ui := cfg.TuiFactory(exitChan)

	// Connect to SSE to receive logs
	go func() {
		resp, err := http.Get("http://localhost:3031/sse")
		if err != nil {
			// Failed to connect, log locally
			cfg.Logger("Failed to connect to SSE logs:", err)
			return
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			if strings.HasPrefix(line, "data: ") {
				msg := strings.TrimPrefix(line, "data: ")
				cfg.Logger(strings.TrimSpace(msg))
			}
		}
	}()

	// Start TUI
	var wg sync.WaitGroup
	wg.Add(1)
	go ui.Start(&wg)
	wg.Wait()
}

func isPortOpen(port string) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", port), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		conn.Close()
		return true
	}
	return false
}

func startDaemonProcess(dir string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Start detached process
	cmd := exec.Command(exe, "-mcp")
	cmd.Dir = dir

	logPath := filepath.Join(dir, "tinywasm-daemon.log")
	logFile, err := os.Create(logPath)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	return cmd.Start()
}

func startSSEServer() *sse.Server {
	s := sse.New(&sse.Config{})
	srv := s.Server(&sse.ServerConfig{})

	go func() {
		http.Handle("/sse", srv)
		if err := http.ListenAndServe(":3031", nil); err != nil {
			fmt.Printf("SSE Server error: %v\n", err)
		}
	}()

	return srv
}
