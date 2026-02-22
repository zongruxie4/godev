package test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDeployUnobservedFilesNotProcessedByAssetmin replicates the exact bug scenario.
func TestDeployUnobservedFilesNotProcessedByAssetmin(t *testing.T) {
	tmp := t.TempDir()

	// Create exact project structure from tinywasm/example
	directories := []string{
		"cmd/appserver",
		"cmd/webclient",
		"cmd/edgeworker",
		"web/public",
		"web/ui/js",
		"deploy/edgeworker",
		"deploy/appserver",
	}

	for _, dir := range directories {
		if err := os.MkdirAll(filepath.Join(tmp, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create a simple edgeworker main.go
	edgeWorkerMain := `package main

func main() {
	println("Edge Worker")
}`
	if err := os.WriteFile(
		filepath.Join(tmp, "cmd/edgeworker/main.go"),
		[]byte(edgeWorkerMain),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	// Create a simple JS file in web/ui that should be processed
	themeJS := `console.log("Theme JS - should be in main.js");`
	if err := os.WriteFile(
		filepath.Join(tmp, "web/ui/theme.js"),
		[]byte(themeJS),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	// Create go.mod
	goMod := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// ⚠️ CRITICAL: Create _worker.js BEFORE starting tinywasm
	workerJsPath := filepath.Join(tmp, "deploy/edgeworker/_worker.js")
	workerContent := `// This is _worker.js content from goflare
export default {
	async fetch(request, env, ctx) {
		return new Response("Edge Worker Response");
	}
};`
	if err := os.WriteFile(workerJsPath, []byte(workerContent), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := startTestApp(t, tmp)
	defer ctx.Cleanup()

	h := ctx.Handler

	// Wait for initialization
	time.Sleep(100 * time.Millisecond)

	// Verify goflare app.Handler exists and has correct UnobservedFiles
	if h == nil {
		t.Fatal("ActiveHandler should be set")
	}
	if h.DeployManager == nil {
		t.Fatal("DeployManager should be initialized")
	}

	unobservedFiles := h.DeployManager.UnobservedFiles()

	// Verify UnobservedFiles contains the expected files (both should be RELATIVE paths)
	expectedFiles := []string{
		"deploy/edgeworker/app.wasm",
		"deploy/edgeworker/_worker.js",
	}

	for _, expectedFile := range expectedFiles {
		found := false
		for _, actual := range unobservedFiles {
			normalizedActual := filepath.ToSlash(actual)
			normalizedExpected := filepath.ToSlash(expectedFile)
			if normalizedActual == normalizedExpected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("UnobservedFiles should contain: %s", expectedFile)
		}
	}

	// Verify paths are relative (not absolute)
	for _, path := range unobservedFiles {
		if filepath.IsAbs(path) {
			t.Fatalf("UnobservedFiles should contain relative paths, got: %s", path)
		}
	}

	// File _worker.js was already created BEFORE starting tinywasm
	if _, err := os.Stat(workerJsPath); os.IsNotExist(err) {
		t.Fatal("_worker.js should exist from pre-creation")
	}

	// Give time for initial registration to complete
	time.Sleep(500 * time.Millisecond)

	// Check main.js path
	mainJsPath := filepath.Join(tmp, "web/public/main.js")

	// Trigger a JS file modification to ensure main.js is written
	themeJsPath := filepath.Join(tmp, "web/ui/theme.js")
	if err := os.WriteFile(themeJsPath, []byte(themeJS+"// modified"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Read main.js if it exists
	var mainJsContent []byte
	if _, err := os.Stat(mainJsPath); err == nil {
		mainJsContent, err = os.ReadFile(mainJsPath)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Parse logs to check if ASSETS app.Handler processed _worker.js
	logsStr := ctx.Logs.String()
	assetLines := []string{}
	for _, line := range strings.Split(logsStr, "\n") {
		if strings.Contains(line, "ASSETS") && strings.Contains(line, ".js") {
			assetLines = append(assetLines, line)
		}
	}

	// THE CRITICAL ASSERTION: _worker.js should NOT be processed by ASSETS
	workerProcessedByAssets := false
	for _, line := range assetLines {
		if strings.Contains(line, "_worker.js") {
			workerProcessedByAssets = true
			t.Errorf("BUG DETECTED: ASSETS app.Handler processed _worker.js: %s", line)
		}
	}

	// Verify _worker.js content is NOT in main.js
	if len(mainJsContent) > 0 {
		workerSignatures := []string{
			"Edge Worker Response",
			"async fetch(request, env, ctx)",
			"export default",
		}

		for _, signature := range workerSignatures {
			if bytes.Contains(mainJsContent, []byte(signature)) {
				t.Errorf("BUG DETECTED: main.js contains _worker.js signature: '%s'", signature)
			}
		}
	}

	// Stop the application handled by defer ctx.Cleanup()
	time.Sleep(100 * time.Millisecond)

	if workerProcessedByAssets {
		t.Fatalf("TEST FAILED: _worker.js was incorrectly processed by ASSETS app.Handler")
	}
}
