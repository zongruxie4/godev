package test

import (
	"testing"
	"github.com/tinywasm/assetmin"
	"github.com/tinywasm/js"
	"strings"
	"os"
)

func TestPageBundle_ContainsBootstrap(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "bundle-test-*")
	defer os.RemoveAll(tmpDir)

	publicDir := tmpDir + "/public"
	os.MkdirAll(publicDir, 0755)

	ah := assetmin.NewAssetMin(&assetmin.Config{
		OutputDir: publicDir,
		RootDir:   tmpDir,
	})

	// Register bootstrap
	ah.UpdateSSRModule("bootstrap", "", []*js.Script{js.PageBootstrap()}, "", nil)

	// Flush to disk to produce script.js
	err := ah.FlushToDisk()
	if err != nil {
		t.Fatalf("FlushToDisk failed: %v", err)
	}

	// Verify script.js exists and contains bootstrap code
	scriptPath := publicDir + "/script.js"
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read script.js: %v", err)
	}

	if !strings.Contains(string(content), "WebAssembly.instantiateStreaming") {
		t.Errorf("script.js does not contain bootstrap code: %s", string(content))
	}

	if !strings.Contains(string(content), "fetch(\"/client.wasm\")") {
		t.Errorf("script.js does not contain fetch(/client.wasm): %s", string(content))
	}
}
