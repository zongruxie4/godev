# VS Code MCP JSON Auto-Configuration

## Overview
Automatically detect and configure VS Code's MCP (Model Context Protocol) integration for GoLite during installation. Similar to TinyWasm's `vscode_config.go` approach for WASM environment setup, this feature ensures GoLite's MCP server is automatically registered in VS Code's configuration without manual user intervention.

## Problem Statement
Users installing GoLite must manually edit `~/.config/Code/User/profiles/[profile-id]/mcp.json` to add:
```json
"golite-mcp": {
  "url": "http://localhost:7070/mcp",
  "type": "http"
}
```

This creates friction after installing GoLite via `go install github.com/cdvelop/golite/cmd/golite@latest` and may prevent users from discovering GoLite's MCP capabilities.

## Requirements

### Primary Goals
1. **Silent Auto-Detection**: Detect VS Code installation without user prompts
2. **Auto-Configuration**: Add/update GoLite MCP configuration in `mcp.json`
3. **Zero User Intervention**: Work seamlessly on first GoLite execution after `go install`
4. **Cross-Platform Support**: Linux, macOS, Windows
5. **Non-Invasive**: Fail silently if VS Code not found or permissions denied

### Constraints
- Execute on first GoLite startup (in `main()` before `ServeMCP()`)
- Do NOT create directories if they don't exist
- Do NOT prompt user for permissions
- Do NOT create backup files
- Update existing `golite-mcp` entry if configuration changes
- Ignore profiles if multiple profiles detected (ambiguity prevention)

## Technical Specification

### Entry Point
Execute in `cmd/golite/main.go` after `golite.Start()` and before `ServeMCP()`:
```go
// Auto-configure VS Code MCP integration (silent)
golite.ConfigureVSCodeMCP()

// Start MCP HTTP server on port 7070 (go-go!)
go golite.ActiveHandler.ServeMCP()
```

### VS Code Detection Logic

#### Platform-Specific Paths
| OS      | Configuration Path |
|---------|-------------------|
| Linux   | `~/.config/Code/User/` |
| macOS   | `~/Library/Application Support/Code/User/` |
| Windows | `%APPDATA%\Code\User\` |

#### Profile Detection Rules
1. Check if base path exists → if NO, return silently
2. Look for `profiles/` subdirectory
3. Count subdirectories in `profiles/`
   - **0 profiles**: Check for `mcp.json` in base path
   - **1 profile**: Use that profile's `mcp.json`
   - **2+ profiles**: Return silently (ambiguous)

### Configuration Structure

#### Target JSON Format
```json
{
  "servers": {
    "golite-mcp": {
      "url": "http://localhost:7070/mcp",
      "type": "http"
    }
  },
  "inputs": []
}
```

#### Port Configuration
- **Constant Location**: Define in `mcp.go` alongside `mcpPort`
- **Current Value**: `7070` (fixed port: go-go!)
- **Future Flexibility**: Centralized definition allows easy updates

### Implementation Steps

#### Step 1: Platform Detection
```go
func getVSCodeConfigPath() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    
    switch runtime.GOOS {
    case "linux":
        return filepath.Join(homeDir, ".config/Code/User"), nil
    case "darwin":
        return filepath.Join(homeDir, "Library/Application Support/Code/User"), nil
    case "windows":
        appData := os.Getenv("APPDATA")
        if appData == "" {
            return "", errors.New("APPDATA not set")
        }
        return filepath.Join(appData, "Code", "User"), nil
    default:
        return "", errors.New("unsupported platform")
    }
}
```

#### Step 2: Profile Resolution
```go
func findMCPConfigPath(basePath string) (string, error) {
    // Check if VS Code User directory exists
    if _, err := os.Stat(basePath); os.IsNotExist(err) {
        return "", errors.New("VS Code not installed")
    }
    
    profilesPath := filepath.Join(basePath, "profiles")
    
    // Check if profiles directory exists
    if _, err := os.Stat(profilesPath); os.IsNotExist(err) {
        // No profiles, use base path
        return filepath.Join(basePath, "mcp.json"), nil
    }
    
    // Count profiles
    entries, err := os.ReadDir(profilesPath)
    if err != nil {
        return "", err
    }
    
    profiles := []string{}
    for _, entry := range entries {
        if entry.IsDir() {
            profiles = append(profiles, entry.Name())
        }
    }
    
    // Only proceed if exactly one profile
    if len(profiles) != 1 {
        return "", errors.New("ambiguous profile count")
    }
    
    return filepath.Join(profilesPath, profiles[0], "mcp.json"), nil
}
```

#### Step 3: JSON Manipulation
```go
type MCPConfig struct {
    Servers map[string]ServerConfig `json:"servers"`
    Inputs  []interface{}           `json:"inputs"`
}

type ServerConfig struct {
    URL  string `json:"url,omitempty"`
    Type string `json:"type"`
    // Other fields for different server types
}

func updateMCPConfig(configPath string, mcpPort string) error {
    var config MCPConfig
    
    // Read existing config
    data, err := os.ReadFile(configPath)
    if err != nil {
        if os.IsNotExist(err) {
            // Create new config
            config = MCPConfig{
                Servers: make(map[string]ServerConfig),
                Inputs:  []interface{}{},
            }
        } else if os.IsPermission(err) {
            // No permissions, return silently
            return nil
        } else {
            return err
        }
    } else {
        // Parse existing config
        if err := json.Unmarshal(data, &config); err != nil {
            return err
        }
        
        if config.Servers == nil {
            config.Servers = make(map[string]ServerConfig)
        }
    }
    
    // Add/update GoLite MCP entry
    config.Servers["golite-mcp"] = ServerConfig{
        URL:  fmt.Sprintf("http://localhost:%s/mcp", mcpPort),
        Type: "http",
    }
    
    // Marshal with proper formatting
    updatedData, err := json.MarshalIndent(config, "", "\t")
    if err != nil {
        return err
    }
    
    // Write back (fail silently on permission errors)
    if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
        if os.IsPermission(err) {
            return nil // Silent failure
        }
        return err
    }
    
    return nil
}
```

#### Step 4: Public API
```go
// ConfigureVSCodeMCP attempts to automatically configure VS Code's MCP integration.
// This function is silent and non-blocking - it will not produce errors or logs.
// Inspired by TinyWasm's VisualStudioCodeWasmEnvConfig approach.
func ConfigureVSCodeMCP() {
    // Get platform-specific VS Code path
    basePath, err := getVSCodeConfigPath()
    if err != nil {
        return // Silent failure
    }
    
    // Resolve profile (or base path)
    configPath, err := findMCPConfigPath(basePath)
    if err != nil {
        return // Silent failure
    }
    
    // Update configuration
    _ = updateMCPConfig(configPath, "7070")
}
```

### Error Handling Strategy

| Error Condition | Behavior |
|----------------|----------|
| VS Code not installed | Return silently |
| Multiple profiles detected | Return silently |
| Permission denied (read) | Return silently |
| Permission denied (write) | Return silently |
| Invalid JSON in existing file | Return silently |
| Unsupported OS | Return silently |

**Rationale**: Configuration assistance is a convenience feature, not a requirement. GoLite functions normally without VS Code integration.

## Testing Strategy

### Test Cases
1. **First Run After go install**
   - VS Code installed, no `mcp.json` → Create file with GoLite config
   
2. **Existing Configuration**
   - `mcp.json` exists with other servers → Add GoLite entry
   - `mcp.json` exists with old GoLite config → Update to latest
   
3. **Edge Cases**
   - No VS Code installed → Silent no-op
   - Multiple profiles → Silent no-op
   - No write permissions → Silent no-op
   - Corrupted JSON → Silent no-op

4. **Cross-Platform**
   - Linux (Ubuntu, Fedora)
   - macOS (Intel, Apple Silicon)
   - Windows (10, 11)

### Test Implementation Pattern
Follow TinyWasm's testing approach:
- Create temporary directories simulating VS Code structure
- Test profile detection logic
- Verify JSON manipulation preserves existing entries
- Validate cross-platform path handling

## Implementation Checklist

- [ ] Create `vscode_config.go` in golite package
- [ ] Implement platform-specific path detection
- [ ] Implement profile resolution logic
- [ ] Implement JSON read/write with merge logic
- [ ] Add port constant to `mcp.go`
- [ ] Integrate into `main.go` (before `ServeMCP()`)
- [ ] Write unit tests for each platform
- [ ] Write integration tests for profile detection
- [ ] Test on Linux, macOS, Windows
- [ ] Update installation documentation

## Future Enhancements

### Potential Improvements (Out of Scope)
1. **Auto-detection of port conflicts**: Scan for available ports if 7070 is occupied
2. **Configuration validation**: Verify MCP server is reachable after setup
3. **Profile selection heuristic**: Use most recently modified profile when multiple exist
4. **Extension recommendation**: Suggest MCP-related VS Code extensions

## References

### Similar Implementations
- **TinyWasm**: `vscode_config.go` - Auto-configures WASM environment variables
- **VS Code Settings**: Uses similar JSON merge pattern for `settings.json`

### VS Code Configuration Locations
- [VS Code User Settings](https://code.visualstudio.com/docs/getstarted/settings)
- [MCP Protocol Specification](https://github.com/modelcontextprotocol/specification)

### Related GoLite Files
- `mcp.go`: MCP server implementation and port definition
- `cmd/golite/main.go`: Entry point for configuration call

### Installation Method
Users install GoLite with:
```bash
go install -v github.com/cdvelop/golite/cmd/golite@latest
```
Configuration happens automatically on first execution.

## Questions & Clarifications

### Resolved Design Decisions
✓ **When to execute**: During main() initialization, before ServeMCP()  
✓ **Multiple profiles**: Ignore (ambiguous)  
✓ **Existing entry**: Update/replace  
✓ **Port configuration**: Centralized constant  
✓ **Error handling**: Silent failures  
✓ **Cross-platform**: All three platforms  
✓ **Logging**: None (completely silent)  
✓ **Backups**: Not created  
✓ **Directory creation**: Not performed  

### Implementation Notes
- Pattern follows TinyWasm's approach: automatic, silent, helpful
- Zero dependencies beyond standard library
- Complements manual MCP setup (doesn't replace documentation)
- Users can still manually configure if auto-detection fails

---

**Document Status**: Ready for Implementation  
**Estimated Complexity**: Medium (similar to TinyWasm's vscode_config.go)  
**Breaking Changes**: None (additive feature)
