package test

import (
	"fmt"
	"testing"

	"github.com/tinywasm/app"
)

func TestSSEPublisherRecentLogs(t *testing.T) {
	pub := app.NewSSEPublisher(nil) // No hub needed for ring buffer test

	// Publish 150 entries
	for i := 1; i <= 150; i++ {
		pub.PublishTabLog("TAB", "HAND", "COL", fmt.Sprintf("msg %d", i))
	}

	logs := pub.RecentLogs()
	if len(logs) != 100 {
		t.Errorf("Expected 100 logs, got %d", len(logs))
	}

	// Should be logs from 51 to 150
	if logs[0] != "msg 51" {
		t.Errorf("First log should be 'msg 51', got '%s'", logs[0])
	}
	if logs[99] != "msg 150" {
		t.Errorf("Last log should be 'msg 150', got '%s'", logs[99])
	}
}
