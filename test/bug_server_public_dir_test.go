package test

// BUG REGRESSION: server_public_dir not passed to external server
//
// When the external server binary runs from its deploy directory, the default
// relative path "web/public" resolves incorrectly. The fix is to pass
// -server_public_dir=<absolute path> in SetRunArgs alongside -server_port.
//
// Repro: running goflare-demo with web/server.go present serves wrong content.
//        Deleting web/server.go (internal mode) works correctly because the
//        internal server is started with the absolute root dir already set.

import (
	"strings"
	"testing"

	"github.com/tinywasm/app"
)

// spyServerConfigurator wraps MockServer and also implements serverConfigurator
// so the type assertion in InitBuildHandlers succeeds and we can capture args.
type spyServerConfigurator struct {
	MockServer
	runArgsFn func() []string
}

func (s *spyServerConfigurator) SetRunArgs(fn func() []string) {
	s.runArgsFn = fn
}

func (s *spyServerConfigurator) SetCompileArgs(fn func() []string) {}

func (s *spyServerConfigurator) SetAppRootDir(dir string) {}

func (s *spyServerConfigurator) SetSourceDir(dir string) {}

func (s *spyServerConfigurator) SetOutputDir(dir string) {}

func (s *spyServerConfigurator) SetMainInputFile(f string) {}

func (s *spyServerConfigurator) SetPort(p string) {}

func (s *spyServerConfigurator) SetDisableGlobalCleanup(v bool) {}

// TestInitBuildHandlers_ExternalServer_PassesPublicDir verifies that when
// InitBuildHandlers wires the external server, the RunArgs function includes
// -server_public_dir with the project's absolute public path.
//
// Without the fix, RunArgs only contains -server_port and the external server
// defaults to the relative "web/public" path, which resolves incorrectly when
// the binary runs from its deploy directory.
func TestInitBuildHandlers_ExternalServer_PassesPublicDir(t *testing.T) {
	tmp := t.TempDir()

	spy := &spyServerConfigurator{}

	h := NewTestHandler(tmp)
	h.RootDir = tmp
	h.GoModHandler = &MockGoModHandler{}
	h.Logger = func(...any) {}
	h.Tui = newUiMockTest()
	h.DB = &MockDB{data: map[string]string{}}
	h.SetServerFactory(func(exitChan chan bool, ui app.TuiInterface, browser app.BrowserInterface) app.ServerInterface {
		return spy
	})
	h.InitBuildHandlers()

	if spy.runArgsFn == nil {
		t.Fatal("SetRunArgs was never called — serverConfigurator interface not satisfied")
	}

	args := spy.runArgsFn()

	// Must contain -server_port
	hasPort := false
	hasPublicDir := false
	for _, a := range args {
		if strings.HasPrefix(a, "-server_port=") {
			hasPort = true
		}
		if strings.HasPrefix(a, "-server_public_dir=") {
			hasPublicDir = true
			// The path must be absolute (contains the tmp dir)
			val := strings.TrimPrefix(a, "-server_public_dir=")
			if !strings.Contains(val, tmp) {
				t.Errorf("-server_public_dir must be absolute and reference the project root; got %q, root is %q", val, tmp)
			}
		}
	}

	if !hasPort {
		t.Error("RunArgs missing -server_port")
	}

	if !hasPublicDir {
		t.Errorf("BUG: RunArgs missing -server_public_dir — external server will default to relative 'web/public' and serve wrong content\nGot args: %v", args)
	}
}
