package app

// ConfigureVSCodeMCP attempts to automatically configure VS Code's MCP integration.
// This function is completely silent and non-blocking - it will not produce errors or logs.
// It detects VS Code installation, resolves all profiles, and updates mcp.json files
// with TinyWasm's MCP server configuration.
//
// Behavior:
//   - Detects platform-specific VS Code paths (Linux, macOS, Windows)
//   - Updates ALL profiles (handles multiple profiles by updating all)
//   - Creates or updates mcp.json with golite-mcp entry in each profile
//   - Fails silently on any error (VS Code not found, permissions, etc.)
//
// Inspired by TinyWasm's VisualStudioCodeWasmEnvConfig approach.
func ConfigureVSCodeMCP() {
	// Get platform-specific VS Code path
	basePath, err := getVSCodeConfigPath()
	if err != nil {
		return // Silent failure: unsupported platform or missing home dir
	}

	// Resolve all profile paths (or base path)
	configPaths, err := findMCPConfigPaths(basePath)
	if err != nil {
		return // Silent failure: VS Code not installed or no profiles
	}

	// Update configuration in all profiles
	for _, configPath := range configPaths {
		_ = updateMCPConfig(configPath, MCPPort)
	}
}
