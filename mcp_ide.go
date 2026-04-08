package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// ConfigureIDEs writes the MCP configuration for various IDEs (VSCode, Cursor, etc.)
// so they can automatically discover the local MCP server.
func ConfigureIDEs(appName, version, port, apiKey string) error {
	// 1. Prepare the MCP server config entry
	configEntry := map[string]any{
		"command": "npx",
		"args": []string{
			"-y",
			"@modelcontextprotocol/inspector",
			"http://localhost:" + port + "/mcp",
		},
		"env": map[string]string{
			"TINYWASM_API_KEY": apiKey,
		},
	}

	// Support for Cursor and VSCode
	ideConfigs := []struct {
		name string
		path string
	}{
		{"Cursor", getIDEConfigPath("Cursor")},
		{"VSCode", getIDEConfigPath("Code")},
	}

	for _, ide := range ideConfigs {
		if ide.path == "" {
			continue
		}

		if err := updateMCPConfigFile(ide.path, appName, configEntry); err != nil {
			// Log error but continue with other IDEs
			continue
		}
	}

	return nil
}

func getIDEConfigPath(ideDirName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	var path string
	switch runtime.GOOS {
	case "darwin":
		path = filepath.Join(home, "Library", "Application Support", ideDirName, "User", "globalStorage", "mcpServers.json")
	case "windows":
		path = filepath.Join(os.Getenv("APPDATA"), ideDirName, "User", "globalStorage", "mcpServers.json")
	case "linux":
		path = filepath.Join(home, ".config", ideDirName, "User", "globalStorage", "mcpServers.json")
	default:
		return ""
	}
	return path
}

func updateMCPConfigFile(path, appName string, configEntry map[string]any) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return err // IDE not installed or storage dir not created
	}

	// Read existing config or create new one
	mcpConfig := make(map[string]any)
	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, &mcpConfig)
	}

	// Ensure mcpServers map exists
	servers, ok := mcpConfig["mcpServers"].(map[string]any)
	if !ok {
		servers = make(map[string]any)
		mcpConfig["mcpServers"] = servers
	}

	// Add or update TinyWasm entry
	servers[appName] = configEntry

	// Write back
	newData, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, newData, 0644)
}
