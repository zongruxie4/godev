package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinywasm/app"
	"github.com/tinywasm/context"
	twjson "github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/fmt"
)

type testToolProvider struct{}

func (tp *testToolProvider) Tools() []mcp.Tool {
	return []mcp.Tool{{
		Name:        "test_tool",
		Description: "Test tool",
		InputSchema: `{"type":"object"}`,
		Resource:    "test",
		Action:      'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("OK"), nil
		},
	}}
}

func TestProjectProxy_ToolsAppearInMCPServerReal(t *testing.T) {
	// Create mcp.Server with an empty proxy (simulates daemon startup)
	proxy := app.NewProjectToolProxy()
	mcpServer, _ := mcp.NewServer(mcp.Config{
		Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer(),
	}, []mcp.ToolProvider{proxy})

	ctx := context.Background()
	ctx.Set(mcp.CtxKeyAuthToken, "test-token")

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/list","params":{}}`)

	// Initially should not have test_tool
	resp1 := mcpServer.HandleMessage(ctx, req)
	var out1 []byte
	if f, ok := resp1.(fmt.Fielder); ok {
		twjson.Encode(f, &out1)
	}
	if strings.Contains(string(out1), "test_tool") {
		t.Fatal("Did not expect test_tool in response yet")
	}

	// Simulate onProjectReady: SetActive + AddTool loop
	tp := &testToolProvider{}
	proxy.SetActive(tp)
	for _, tool := range tp.Tools() {
		mcpServer.AddTool(tool)
	}

	// Now tools/list must return the tools from our provider
	resp2 := mcpServer.HandleMessage(ctx, req)
	if resp2 == nil {
		t.Fatal("Expected response from HandleMessage, got nil")
	}

	var out2 []byte
	if f, ok := resp2.(fmt.Fielder); ok {
		twjson.Encode(f, &out2)
	}
	if !strings.Contains(string(out2), "test_tool") {
		t.Fatalf("Expected test_tool in response, got: %s", string(out2))
	}
}

func TestHandlerTools_AppRebuildNotExposed(t *testing.T) {
	h := &app.Handler{} // zero-value Handler
	tools := h.Tools()
	for _, tool := range tools {
		if tool.Name == "app_rebuild" {
			t.Fatalf("app_rebuild must not be exposed as an MCP tool, found in Handler.Tools()")
		}
	}
}

func TestDaemonAction_QuitClosesExitChan(t *testing.T) {
	exitChan := make(chan bool)
	closed := make(chan struct{})

	// Spin up a minimal HTTP server that handles POST /tinywasm/action
	// with the same logic as daemon.go but wired to exitChan
	mux := http.NewServeMux()
	var once sync.Once
	mux.HandleFunc("POST /tinywasm/action", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		key := string(mcp.ExtractJSONValue(body, "key"))   // unquote not needed for plain string
		if string(key) == `"quit"` || string(key) == "quit" {
			once.Do(func() { close(exitChan) })
		}
		w.Write([]byte("OK"))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Goroutine that watches exitChan
	go func() {
		<-exitChan
		close(closed)
	}()

	// Client sends quit
	http.Post(srv.URL+"/tinywasm/action",
		"application/json",
		strings.NewReader(`{"key":"quit"}`))

	select {
	case <-closed:
		// pass
	case <-time.After(2 * time.Second):
		t.Fatal("exitChan was not closed after quit action")
	}
}
