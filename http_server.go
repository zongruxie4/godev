package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ssePublisher is the DI interface for SSE transport.
// Implemented by tinywasm/sse.SSEServer.
type ssePublisher interface {
	http.Handler
	Publish(data []byte, channel string)
	PublishEvent(event string, data []byte, channels ...string)
}

// LogEntry is the wire format for SSE log events consumed by devtui/sse_client.go.
// JSON field names MUST match tabContentDTO in devtui — do not rename.
type LogEntry struct {
	Id           string `json:"id"`
	Timestamp    string `json:"timestamp"`
	Content      string `json:"content"`
	Type         uint8  `json:"type"`
	TabTitle     string `json:"tab_title"`
	HandlerName  string `json:"handler_name"`
	HandlerColor string `json:"handler_color"`
	HandlerType  int    `json:"handler_type"`
}

// TinywasmHTTP is the HTTP server for the TinyWASM daemon and standalone mode.
// It owns the http.Server, SSE hub, and all tinywasm-specific routes.
// Implements Name() + SetLog() so it can be registered as a TUI handler.
type TinywasmHTTP struct {
	port       string
	sseHub     ssePublisher  // interface: Publish(data, channel) + http.Handler
	mcpHTTP    http.Handler  // from mcp.Handler.HTTPHandler()
	onAction   func(key, value string)
	onState    func() []byte
	appVersion string
	log        func(messages ...any)
	server     *http.Server
	mu         sync.Mutex
}

// NewTinywasmHTTP creates the HTTP server with all routes pre-configured.
func NewTinywasmHTTP(port string, mcpHTTP http.Handler, sseHub ssePublisher, appVersion string) *TinywasmHTTP {
	return &TinywasmHTTP{
		port:       port,
		mcpHTTP:    mcpHTTP,
		sseHub:     sseHub,
		appVersion: appVersion,
		log:        func(messages ...any) {},
	}
}

// OnAction registers the callback for POST /action?key=...
func (s *TinywasmHTTP) OnAction(fn func(key, value string)) {
	s.mu.Lock()
	s.onAction = fn
	s.mu.Unlock()
}

// OnState registers the callback for GET /state
func (s *TinywasmHTTP) OnState(fn func() []byte) {
	s.mu.Lock()
	s.onState = fn
	s.mu.Unlock()
}

// SetLog satisfies the Loggable interface — app registers TinywasmHTTP in the TUI.
func (s *TinywasmHTTP) SetLog(fn func(messages ...any)) {
	s.mu.Lock()
	s.log = fn
	s.mu.Unlock()
}

// Name satisfies the Loggable interface — returns "MCP" for TUI display.
func (s *TinywasmHTTP) Name() string {
	return "MCP"
}

// Serve starts the HTTP server and blocks until exitChan is closed.
func (s *TinywasmHTTP) Serve(exitChan chan bool) {
	mux := http.NewServeMux()
	mux.Handle("/mcp", s.mcpHTTP)
	if s.sseHub != nil {
		mux.Handle("/logs", s.sseHub)
	}
	mux.HandleFunc("/action", s.handleActionPOST)
	mux.HandleFunc("/state", s.handleStateGET)
	mux.HandleFunc("/version", s.handleVersion)

	s.mu.Lock()
	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}
	s.mu.Unlock()

	s.log("Started on :" + s.port + "/mcp")

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("HTTP server stopped:", err)
		}
	}()

	// Wait for exit signal
	<-exitChan
	s.Stop()
}

// Stop gracefully shuts down the HTTP server.
func (s *TinywasmHTTP) Stop() {
	s.mu.Lock()
	srv := s.server
	s.mu.Unlock()

	if srv == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.log("Error shutting down HTTP server:", err)
	}
}

func (s *TinywasmHTTP) handleActionPOST(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "query param 'key' is required", http.StatusBadRequest)
		return
	}
	value := r.URL.Query().Get("value")

	s.mu.Lock()
	actionCb := s.onAction
	s.mu.Unlock()

	if actionCb != nil {
		actionCb(key, value)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Action applied: %s", key)))
	} else {
		http.Error(w, "No action handler configured", http.StatusServiceUnavailable)
	}
}

func (s *TinywasmHTTP) handleStateGET(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.onState
	s.mu.Unlock()

	if fn == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(fn())
}

func (s *TinywasmHTTP) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"version":"` + s.appVersion + `"}`))
}

// PublishTabLog publishes a structured log entry to the SSE stream.
func (s *TinywasmHTTP) PublishTabLog(tabTitle, handlerName, handlerColor, msg string) {
	if s.sseHub == nil {
		return
	}
	entry := LogEntry{
		Id:           fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp:    time.Now().Format("15:04:05"),
		Content:      msg,
		Type:         1,
		TabTitle:     tabTitle,
		HandlerName:  handlerName,
		HandlerColor: handlerColor,
		HandlerType:  4, // HandlerTypeLoggable
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	s.sseHub.Publish(data, "logs")
}

// PublishLog publishes a plain log to the BUILD tab under "MCP" handler.
func (s *TinywasmHTTP) PublishLog(msg string) {
	s.PublishTabLog("BUILD", "MCP", "#f97316", msg)
}

// PublishStateEvent broadcasts a typed "state" SSE event so connected clients
// can reconstruct their remote handler footer fields immediately.
func (s *TinywasmHTTP) PublishStateEvent(stateJSON []byte) {
	if s.sseHub == nil {
		return
	}
	s.sseHub.PublishEvent("state", stateJSON, "logs")
}
