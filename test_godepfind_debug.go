package godev

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cdvelop/godepfind"
	"github.com/cdvelop/goserver"
	"github.com/stretchr/testify/require"
)

// TestGodepfindServerDetection specifically tests if godepfind can correctly identify server files
func TestGodepfindServerDetection(t *testing.T) {
	tmp := t.TempDir()

	// Create the exact structure needed for server detection
	pwaDir := filepath.Join(tmp, "pwa")
	require.NoError(t, os.MkdirAll(pwaDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(pwaDir, "public"), 0755))

	// Create go.mod to establish module context
	goModContent := `module testproject

go 1.20
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

	// Create main.server.go file
	serverFilePath := filepath.Join(pwaDir, "main.server.go")
	serverContent := `package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello")
	})
	http.ListenAndServe(":8080", nil)
}
`
	require.NoError(t, os.WriteFile(serverFilePath, []byte(serverContent), 0644))

	// Create a ServerHandler like godev does
	serverHandler := goserver.New(&goserver.Config{
		RootFolder:                  "pwa",
		MainFileWithoutExtension:    "main.server",
		ArgumentsForCompilingServer: func() []string { return nil },
		ArgumentsToRunServer:        func() []string { return nil },
		PublicFolder:                "public",
		AppPort:                     "4430",
		Logger:                      os.Stdout,
		ExitChan:                    make(chan bool),
	})

	t.Logf("=== TESTING GODEPFIND DETECTION ===")
	t.Logf("Project root: %s", tmp)
	t.Logf("Server file path: %s", serverFilePath)
	t.Logf("ServerHandler.MainFilePath(): %s", serverHandler.MainFilePath())
	t.Logf("ServerHandler.Name(): %s", serverHandler.Name())

	// Create godepfind instance
	depFinder := godepfind.New(tmp)

	// Test different path formats that might be used in real scenario
	testCases := []struct {
		name     string
		fileName string
		filePath string
		event    string
	}{
		{
			name:     "absolute_path_write",
			fileName: "main.server.go",
			filePath: serverFilePath,
			event:    "write",
		},
		{
			name:     "relative_path_write",
			fileName: "main.server.go",
			filePath: "pwa/main.server.go",
			event:    "write",
		},
		{
			name:     "relative_path_create",
			fileName: "main.server.go",
			filePath: "pwa/main.server.go",
			event:    "create",
		},
		{
			name:     "just_filename_write",
			fileName: "main.server.go",
			filePath: "",
			event:    "write",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isMine, err := depFinder.ThisFileIsMine(serverHandler, tc.fileName, tc.filePath, tc.event)
			t.Logf("ThisFileIsMine(fileName='%s', filePath='%s', event='%s') = %v, err: %v",
				tc.fileName, tc.filePath, tc.event, isMine, err)

			if !isMine && err == nil {
				t.Logf("  ❌ NOT DETECTED - This could be the problem!")
			} else if isMine {
				t.Logf("  ✅ DETECTED - Working correctly")
			} else {
				t.Logf("  ❓ ERROR: %v", err)
			}
		})
	}

	// Test what happens when we change working directory (like in real scenario)
	t.Run("with_changed_workdir", func(t *testing.T) {
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)

		// Change to project directory like real usage
		os.Chdir(tmp)

		// Test relative paths from project root
		relServerPath, _ := filepath.Rel(tmp, serverFilePath)
		isMine, err := depFinder.ThisFileIsMine(serverHandler, "main.server.go", relServerPath, "write")
		t.Logf("From project root - ThisFileIsMine('main.server.go', '%s', 'write') = %v, err: %v",
			relServerPath, isMine, err)
	})

	// Test edge cases that might reveal the issue
	t.Run("debug_internal_state", func(t *testing.T) {
		t.Logf("=== DEBUGGING INTERNAL STATE ===")

		// Try to force cache initialization by calling with any file first
		_, _ = depFinder.ThisFileIsMine(serverHandler, "dummy.go", "", "write")

		// Now test our actual file
		isMine, err := depFinder.ThisFileIsMine(serverHandler, "main.server.go", serverFilePath, "write")
		t.Logf("After cache init - isMine: %v, err: %v", isMine, err)

		// Test with the exact path format that the server handler expects
		handlerPath := serverHandler.MainFilePath()
		t.Logf("Handler expects path: %s", handlerPath)

		// Try with just the relative part that matches MainFilePath
		isMineHandler, errHandler := depFinder.ThisFileIsMine(serverHandler, "main.server.go", handlerPath, "write")
		t.Logf("Using handler path - isMine: %v, err: %v", isMineHandler, errHandler)
	})
}
