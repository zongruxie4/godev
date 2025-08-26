package godev

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	gs "github.com/cdvelop/goserver"
)

// TestSimpleServerRestart reproduce un escenario mínimo: arrancar el ServerHandler,
// escribir un archivo con error de compilación, luego corregirlo y disparar eventos
// para comprobar que tras la corrección se reinicia correctamente.
func TestSimpleServerRestart(t *testing.T) {
	tmp := t.TempDir()

	// Create minimal project folder `pwa` to mimic godev layout
	pwa := filepath.Join(tmp, "pwa")
	requireNoErr(t, os.MkdirAll(pwa, 0755))

	// write simple go.mod so `go build` works in that folder
	requireNoErr(t, os.WriteFile(filepath.Join(pwa, "go.mod"), []byte("module temp/pwa\n\n go 1.20\n"), 0644))

	serverFile := filepath.Join(pwa, "main.server.go")

	initial := `package main

import (
    "fmt"
    "net/http"
    "log"
)

func main() {
    fmt.Println("OK_V1")
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "OK_V1") })
    log.Fatal(http.ListenAndServe(":0", nil))
}`

	requireNoErr(t, os.WriteFile(serverFile, []byte(initial), 0644))

	var logBuf bytes.Buffer

	cfg := &gs.Config{
		RootFolder:               pwa,
		MainFileWithoutExtension: "main.server",
		Logger:                   &logBuf,
		ExitChan:                 make(chan bool, 1),
	}

	handler := gs.New(cfg)

	var wg sync.WaitGroup
	wg.Add(1)
	go handler.StartServer(&wg)
	wg.Wait()

	time.Sleep(300 * time.Millisecond)

	// write broken version
	broken := `package main

import (
    "fmt"
    "net/http"
    "log"
)

func main() {
    fmt.rintf("BROKEN")
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "BROKEN") })
    log.Fatal(http.ListenAndServe(":0", nil))
}`

	requireNoErr(t, os.WriteFile(serverFile, []byte(broken), 0644))

	// Trigger event: expect an error due to compile failure
	if err := handler.NewFileEvent("main.server.go", "go", serverFile, "write"); err == nil {
		t.Fatalf("expected error on restart with broken code")
	}

	time.Sleep(200 * time.Millisecond)

	// fix file
	fixed := `package main

import (
    "fmt"
    "net/http"
    "log"
)

func main() {
    fmt.Println("OK_FIXED")
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "OK_FIXED") })
    log.Fatal(http.ListenAndServe(":0", nil))
}`

	requireNoErr(t, os.WriteFile(serverFile, []byte(fixed), 0644))

	// Trigger event again: should succeed
	if err := handler.NewFileEvent("main.server.go", "go", serverFile, "write"); err != nil {
		t.Fatalf("expected successful restart after fix, got: %v", err)
	}

	time.Sleep(400 * time.Millisecond)

	// stop server
	cfg.ExitChan <- true
	time.Sleep(100 * time.Millisecond)

	// quick check logs exist
	if logBuf.Len() == 0 {
		logIfVerbose(t, "warning: no logs captured; but restart flow executed")
	}
}

// small helper to fail on error
func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}
