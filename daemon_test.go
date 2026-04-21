package app

import (
	stdjson "encoding/json"
	"testing"

	"github.com/tinywasm/fmt"
	twjson "github.com/tinywasm/json"
)

func TestStateResponse_EncodesCorrectly(t *testing.T) {
	stateJSON := []byte(`[{"tab_title":"BUILD","handler_name":"WasmClient","handler_type":1}]`)
	id := `"1"`

	sr := stateResponse{
		JSONRPC: "2.0",
		ID:      fmt.RawJSON(id),
		Result:  fmt.RawJSON(string(stateJSON)),
	}
	var respBytes []byte
	twjson.Encode(&sr, &respBytes)

	// Verify valid JSON with result as array (not string)
	var envelope map[string]any
	if err := stdjson.Unmarshal(respBytes, &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, respBytes)
	}

	arr, ok := envelope["result"].([]any)
	if !ok || len(arr) == 0 {
		t.Fatalf("result must be non-empty array, got: %T %v", envelope["result"], envelope["result"])
	}

	entry := arr[0].(map[string]any)
	if entry["tab_title"] != "BUILD" {
		t.Errorf("tab_title: got %v, want BUILD", entry["tab_title"])
	}

	if envelope["id"] != "1" {
		t.Errorf("id: got %v, want 1", envelope["id"])
	}
}
