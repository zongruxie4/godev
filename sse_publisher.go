package app

import (
	"encoding/json"
	"fmt"
	"sync"
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
type SSEPublisher struct {
	hub   ssePublisher
	mu    sync.Mutex
	ring  [100]string
	head  int
	count int
}

func NewSSEPublisher(hub ssePublisher) *SSEPublisher { return &SSEPublisher{hub: hub} }

func (p *SSEPublisher) addToRing(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ring[p.head] = msg
	p.head = (p.head + 1) % 100
	if p.count < 100 {
		p.count++
	}
}

// RecentLogs returns up to 100 of the latest log entries in chronological order.
func (p *SSEPublisher) RecentLogs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	res := make([]string, 0, p.count)
	if p.count < 100 {
		for i := 0; i < p.count; i++ {
			res = append(res, p.ring[i])
		}
	} else {
		for i := 0; i < 100; i++ {
			res = append(res, p.ring[(p.head+i)%100])
		}
	}
	return res
}

func (p *SSEPublisher) PublishTabLog(tabTitle, handlerName, handlerColor, msg string) {
	p.addToRing(msg)
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
	p.PublishTabLog("MCP", "MCP", colorOrangeLight, msg)
}

// PublishStateRefresh sends a lightweight signal to connected devtui clients
func (p *SSEPublisher) PublishStateRefresh() {
	if p.hub == nil {
		return
	}
	signal, _ := json.Marshal(map[string]any{"handler_type": 0}) // TypeStateRefresh
	p.hub.Publish(signal, "logs")
}
