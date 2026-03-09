package app

import (
	"net/http"

	"github.com/tinywasm/sse"
)

// sseHubAdapter wraps *sse.SSEServer to implement mcp.SSEHub and ssePublisher
type sseHubAdapter struct {
	*sse.SSEServer
}

func (a *sseHubAdapter) Publish(data []byte, channels ...string) {
	for _, channel := range channels {
		a.SSEServer.Publish(data, channel)
	}
}

// logChannelProvider implements sse.ChannelProvider
type logChannelProvider struct{}

func (p *logChannelProvider) ResolveChannels(r *http.Request) ([]string, error) {
	return []string{"logs"}, nil
}
