package golite

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// mcpExecuteTool creates a GENERIC tool executor that works for ANY handler tool
// It extracts args, collects progress, executes the tool, and returns results
// NO domain-specific logic here - handlers provide their own Execute functions
func (h *handler) mcpExecuteTool(executor ToolExecutor) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// 1. Extract arguments (generic)
		args, ok := req.Params.Arguments.(map[string]any)
		if !ok {
			// Handle case where no arguments are provided
			args = make(map[string]any)
		}

		// 2. Create channel and collect messages
		progressChan := make(chan string, 10) // Buffered to avoid blocking
		messages := []string{}
		done := make(chan bool)

		// 3. Collect progress messages in goroutine
		go func() {
			for msg := range progressChan {
				messages = append(messages, msg)
			}
			done <- true
		}()

		// 4. Execute handler-specific logic (sends messages to channel)
		executor(args, progressChan)
		close(progressChan)

		// 5. Wait for collection to finish
		<-done

		// 6. Refresh UI (generic)
		if h.tui != nil {
			h.tui.RefreshUI()
		}

		// 7. Return collected output (generic)
		if len(messages) == 0 {
			return mcp.NewToolResultText("Operation completed successfully"), nil
		}

		return mcp.NewToolResultText(strings.Join(messages, "\n")), nil
	}
}
