package test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/tinywasm/app"
	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
	twjson "github.com/tinywasm/json"
)

func TestDaemonGetLogsFromRing(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "daemon_logs_test")
	defer os.RemoveAll(tmpDir)

	cfg := app.BootstrapConfig{
		Version: "1.0.0",
		Logger:  func(m ...any) {},
	}

	dtp := app.NewDaemonToolProvider(cfg, func(m ...any) {})
	ssePub := app.NewSSEPublisher(nil)
	dtp.SetSSEPub(ssePub)
	dtp.SetLastPath(tmpDir)

	// Add some logs to the ring buffer
	for i := 1; i <= 5; i++ {
		ssePub.PublishTabLog("TAB", "HAND", "COL", fmt.Sprintf("Log line %d", i))
	}

	ctx := context.Background()
	req := mcp.Request{
		Params: mcp.CallToolParams{
			Arguments: `{"lines": 3}`,
		},
	}

	result, err := dtp.ExecuteGetLogs(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteGetLogs failed: %v", err)
	}

	contentJSON := string(result.Content)

	var contentList mcp.TextContentList
	if err := twjson.Decode([]byte(contentJSON), &contentList); err != nil {
		t.Fatalf("Failed to decode result content: %v. Content: %s", err, contentJSON)
	}

	if contentList.Len() != 1 {
		t.Fatalf("Expected 1 content item, got %d", contentList.Len())
	}

	text := contentList[0].Text
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 log lines, got %d", len(lines))
	}
	if !strings.Contains(lines[2], "Log line 5") {
		t.Errorf("Last line should contain 'Log line 5', got '%s'", lines[2])
	}
}
