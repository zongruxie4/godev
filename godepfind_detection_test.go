package godev

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cdvelop/godepfind"
	"github.com/cdvelop/goserver"
	"github.com/stretchr/testify/require"
)

// TestGodepfindDetection ensures godepfind identifies server files in common path formats.
func TestGodepfindDetection(t *testing.T) {
	tmp := t.TempDir()

	pwaDir := filepath.Join(tmp, "pwa")
	require.NoError(t, os.MkdirAll(pwaDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(pwaDir, "public"), 0755))

	goModContent := `module testproject

go 1.20
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(goModContent), 0644))

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

	serverHandler := goserver.New(&goserver.Config{
		RootFolder:                  "pwa",
		MainFileWithoutExtension:    "main.server",
		ArgumentsForCompilingServer: func() []string { return nil },
		ArgumentsToRunServer:        func() []string { return nil },
		PublicFolder:                "public",
		AppPort:                     "4430",
		Logger:                      nil,
		ExitChan:                    make(chan bool),
	})

	depFinder := godepfind.New(tmp)

	cases := []struct {
		name      string
		fileName  string
		filePath  string
		event     string
		expect    bool
		expectErr bool
	}{
		{"absolute_path_write", "main.server.go", serverFilePath, "write", true, false},
		{"relative_path_write", "main.server.go", "pwa/main.server.go", "write", true, false},
		{"relative_path_create", "main.server.go", "pwa/main.server.go", "create", true, false},
		{"just_filename_should_error", "main.server.go", "", "write", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isMine, err := depFinder.ThisFileIsMine(serverHandler, tc.filePath, tc.event)

			if tc.expectErr {
				require.Error(t, err, "expected error for %s", tc.name)
				require.False(t, isMine, "expected isMine to be false when error occurs for %s", tc.name)
			} else {
				require.NoError(t, err)
				if tc.expect {
					require.True(t, isMine, "expected ThisFileIsMine to be true for %s", tc.name)
				} else {
					require.False(t, isMine, "expected ThisFileIsMine to be false for %s", tc.name)
				}
			}
		})
	}

	t.Run("with_changed_workdir", func(t *testing.T) {
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		os.Chdir(tmp)
		relServerPath, _ := filepath.Rel(tmp, serverFilePath)
		isMine, err := depFinder.ThisFileIsMine(serverHandler, relServerPath, "write")
		require.NoError(t, err)
		require.True(t, isMine)
	})
}
