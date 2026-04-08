package app

import (
	"encoding/json"
	"fmt"
	"time"
)

// ssePublisher is the DI interface for SSE transport (tinywasm/sse.SSEServer).
type ssePublisher interface {
	Publish(data []byte, channel string)
}

// LogEntry is the SSE wire format consumed by devtui/sse_client.go.
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

// SSEPublisher wraps an ssePublisher hub with tinywasm-specific publishing logic.
type SSEPublisher struct{ hub ssePublisher }

func NewSSEPublisher(hub ssePublisher) *SSEPublisher { return &SSEPublisher{hub: hub} }

func (p *SSEPublisher) PublishTabLog(tabTitle, handlerName, handlerColor, msg string) {
	if p.hub == nil {
		return
	}
	entry := LogEntry{
		Id:           fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp:    time.Now().Format("15:04:05"),
		Content:      msg, Type: 1,
		TabTitle:     tabTitle,
		HandlerName:  handlerName,
		HandlerColor: handlerColor,
		HandlerType:  4, // HandlerTypeLoggable
	}
	data, _ := json.Marshal(entry)
	p.hub.Publish(data, "logs")
}

func (p *SSEPublisher) PublishLog(msg string) {
	p.PublishTabLog("BUILD", "MCP", "#f97316", msg)
}

// PublishStateRefresh sends a lightweight signal to connected devtui clients
func (p *SSEPublisher) PublishStateRefresh() {
	if p.hub == nil {
		return
	}
	signal, _ := json.Marshal(map[string]any{"handler_type": 0}) // TypeStateRefresh
	p.hub.Publish(signal, "logs")
}
