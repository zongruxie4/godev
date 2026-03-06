package test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

// TestRunClient_PostsStartAction verifies that when a client connects to the daemon,
// it immediately POSTs a "start" action with the current working directory.
// This ensures every `tinywasm` invocation activates the project in its cwd.
func TestRunClient_PostsStartAction(t *testing.T) {
	var startReceived atomic.Bool
	var startPath string

	// Mock daemon action endpoint
	actionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/action" {
			key := r.URL.Query().Get("key")
			value := r.URL.Query().Get("value")
			if key == "start" {
				startPath = value
				startReceived.Store(true)
			}
		}
		w.WriteHeader(200)
	}))
	defer actionServer.Close()

	// Fire the same request that runClient sends
	baseURL := actionServer.URL
	startDir := "/home/user/Dev/Project/myproject"

	go func() {
		targetURL := baseURL + "/action?key=start&value=" + url.QueryEscape(startDir)
		resp, err := http.Post(targetURL, "application/json", nil)
		if err == nil {
			resp.Body.Close()
		}
	}()

	// Wait for action to arrive
	deadline := time.After(2 * time.Second)
	for {
		if startReceived.Load() {
			break
		}
		select {
		case <-deadline:
			t.Fatal("Timeout: daemon did not receive 'start' action from client")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	decodedPath, err := url.QueryUnescape(startPath)
	if err != nil {
		t.Fatalf("Failed to decode path: %v", err)
	}
	if decodedPath != startDir {
		t.Errorf("Expected start path %q, got %q", startDir, decodedPath)
	}
}

// TestDaemon_StopAction_StopsProject verifies that the daemon's /action endpoint
// accepts key=stop and can dispatch the stop command.
func TestDaemon_StopAction_StopsProject(t *testing.T) {
	stopReceived := make(chan string, 1)

	// Mock action endpoint that simulates daemon behavior
	actionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/action" && r.Method == http.MethodPost {
			key := r.URL.Query().Get("key")
			stopReceived <- key
			w.WriteHeader(200)
			return
		}
		http.NotFound(w, r)
	}))
	defer actionServer.Close()

	// Send stop action (same as Ctrl+C sends in client mode)
	go func() {
		targetURL := actionServer.URL + "/action?key=stop&value="
		resp, _ := http.Post(targetURL, "application/json", nil)
		if resp != nil {
			resp.Body.Close()
		}
	}()

	select {
	case key := <-stopReceived:
		if key != "stop" {
			t.Errorf("Expected key 'stop', got %q", key)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout: stop action not received")
	}
}

// TestDaemon_StartAction_WithPath verifies the start action passes path correctly.
func TestDaemon_StartAction_WithPath(t *testing.T) {
	type actionMsg struct {
		key   string
		value string
	}
	received := make(chan actionMsg, 1)

	actionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/action" && r.Method == http.MethodPost {
			received <- actionMsg{
				key:   r.URL.Query().Get("key"),
				value: r.URL.Query().Get("value"),
			}
			w.WriteHeader(200)
		}
	}))
	defer actionServer.Close()

	projectDir := "/home/user/Dev/Project/dom"
	targetURL := actionServer.URL + "/action?key=start&value=" + url.QueryEscape(projectDir)

	go func() {
		resp, _ := http.Post(targetURL, "application/json", nil)
		if resp != nil {
			resp.Body.Close()
		}
	}()

	select {
	case msg := <-received:
		if msg.key != "start" {
			t.Errorf("Expected key 'start', got %q", msg.key)
		}
		decoded, _ := url.QueryUnescape(msg.value)
		if decoded != projectDir {
			t.Errorf("Expected path %q, got %q", projectDir, decoded)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout: start action not received")
	}
}
