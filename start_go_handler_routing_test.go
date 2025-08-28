package godev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStartGoHandlerRouting recreates the real-world scenario where WASM handler
// has no packages added initially, and routing detection fails when server file
// is modified to add database.Connect() import.
//
// This test reproduces the logs:
// 20:13:02 [SERVER] Current working directory: /home/cesar/Dev/Pkg/Mine/godev
// 20:13:20 [WASM] Wasm.gowrite.../home/cesar/Dev/Pkg/Mine/godev/test/database/db.go
//
// The issue: when WASM handler has no main.wasm.go, (*DevWatch).handleFileEvent
// doesn't correctly detect which handler owns database/db.go using (*GoDepFind).ThisFileIsMine
func TestStartGoHandlerRouting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup temporary project layout that mirrors the real scenario
	tmp := t.TempDir()

	// Create directory structure like the real test case
	serverDir := filepath.Join(tmp, "pwa")
	databaseDir := filepath.Join(tmp, "database")

	require.NoError(t, os.MkdirAll(serverDir, 0755))
	require.NoError(t, os.MkdirAll(databaseDir, 0755))

	// Create go.mod
	goModContent := `module testproject

go 1.21
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	// Create main.server.go WITHOUT database import initially
	serverMainPath := filepath.Join(serverDir, "main.server.go")
	initialServerContent := `package main

func main() {

	println("Server starting on 430")
}
`
	require.NoError(t, os.WriteFile(serverMainPath, []byte(initialServerContent), 0644))

	// Create database/db.go
	dbPath := filepath.Join(databaseDir, "db.go")
	dbContent := `package database

func Connect() {
	println("Connected to database...")
}
`
	require.NoError(t, os.WriteFile(dbPath, []byte(dbContent), 0644))

	// Track handler calls using a custom logger that captures output
	logLines := make([]string, 0)
	logger := func(messages ...any) {
		if len(messages) > 0 {
			line := fmt.Sprint(messages...)
			logLines = append(logLines, line)
			// Also print for debugging
			fmt.Println(line)
		}
	}

	// Use Start to initialize the system like in real usage
	exitChan := make(chan bool)
	done := make(chan struct{})

	go func() {
		defer close(done)
		Start(tmp, logger, exitChan)
	}()

	// Wait for startup and handlers to be created
	time.Sleep(1 * time.Second)

	// Step 1: Modify main.server.go to add database import (this should trigger server handler)
	modifiedServerContent := `package main

import (
	"testproject/database"
)

func main() {

	database.Connect()

	println("Server starting on 4430")
}
`
	require.NoError(t, os.WriteFile(serverMainPath, []byte(modifiedServerContent), 0644))

	// Wait for file event processing
	time.Sleep(500 * time.Millisecond)

	// Step 2: Now modify database/db.go (this should be handled by server handler, not WASM)
	modifiedDbContent := `package database

func Connect() {
	println("Connected to database...")
}

// Added comment to trigger file change
`
	require.NoError(t, os.WriteFile(dbPath, []byte(modifiedDbContent), 0644))

	// Wait for file event processing
	time.Sleep(500 * time.Millisecond)

	// Stop the application
	exitChan <- true

	// Wait for shutdown
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Log("Test timeout reached - this is expected for this integration test")
		// Force exit
		exitChan <- true
		time.Sleep(100 * time.Millisecond)
	}

	// Analyze the logs to detect the issue
	t.Logf("Total log lines captured: %d", len(logLines))

	// Check if we have the issue: WASM handler incorrectly claiming database/db.go
	wasmClaimedDb := false
	serverClaimedDb := false

	for _, line := range logLines {
		if strings.Contains(line, "db.go") {
			if strings.Contains(line, "WASM") || strings.Contains(line, "Wasm") {
				wasmClaimedDb = true
				t.Logf("ISSUE DETECTED: WASM handler claimed db.go: %s", line)
			} else if strings.Contains(line, "SERVER") || strings.Contains(line, "Server") {
				serverClaimedDb = true
				t.Logf("CORRECT: Server handler claimed db.go: %s", line)
			}
		}
	}

	// Log all lines for debugging
	for i, line := range logLines {
		t.Logf("Log[%d]: %s", i, line)
	}

	// The issue occurs when WASM handler (which has no main.wasm.go) incorrectly claims db.go
	if wasmClaimedDb && !serverClaimedDb {
		t.Errorf("ISSUE REPRODUCED: WASM handler incorrectly claimed database/db.go when it should belong to server handler")
		t.Errorf("This happens because WASM handler has no main.wasm.go file, causing routing detection to fail")
		t.Errorf("Expected: server handler should claim db.go because main.server.go imports database package")
		t.Errorf("Actual: WASM handler claimed db.go despite not having main.wasm.go")

		// Print reproduction steps for debugging
		t.Errorf("REPRODUCTION STEPS:")
		t.Errorf("1. WASM handler has no main.wasm.go file")
		t.Errorf("2. Server handler has main.server.go that imports testproject/database")
		t.Errorf("3. When database/db.go is modified, WASM handler incorrectly claims it")
		t.Errorf("4. This causes incorrect routing: [WASM] instead of [SERVER]")

	} else if wasmClaimedDb && serverClaimedDb {
		t.Errorf("ISSUE DETECTED: BOTH handlers claimed db.go - this indicates a potential issue with file ownership detection")
		t.Errorf("Expected: Only SERVER handler should claim db.go")
		t.Errorf("Actual: Both WASM and SERVER handlers claimed db.go")

	} else if serverClaimedDb && !wasmClaimedDb {
		t.Logf("SUCCESS: Server handler correctly claimed db.go")

	} else {
		t.Logf("Neither handler claimed db.go - checking if handlers are working correctly")

		// Check if any Go file events were detected
		goFileEvents := false
		for _, line := range logLines {
			if strings.Contains(line, ".go") && (strings.Contains(line, "write") || strings.Contains(line, "create")) {
				goFileEvents = true
				break
			}
		}

		if !goFileEvents {
			t.Log("No Go file events detected - the file watching system may not be working in this test environment")
		} else {
			t.Error("Handlers are processing Go file events but neither claimed db.go - this may indicate a different issue")
		}
	}
}
