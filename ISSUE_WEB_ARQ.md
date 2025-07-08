# ISSUE: Web Architecture Refactoring

## Overview
Refactor the web architecture detection and handling to move from `web/` subdirectories to root-level architecture directories (`pwa/`, `spa/`, `mpa/`). This change eliminates the `web/` folder and simplifies the project structure following convention over configuration.

## Current vs New Structure

### Before (Current)
```
AppName/
├── cmd/
├── modules/
├── web/
│   ├── pwa/
│   ├── spa/
│   ├── theme/
│   └── public/
└── go.mod
```

### After (Target)
```
AppName/
├── cmd/                    # Optional - CLI app
├── modules/                # Required - modular logic
├── pwa/                    # PWA Architecture (priority 1)
│   ├── theme/              # Unprocessed CSS/JS
│   ├── public/             # Final assets + manifest.json + sw.js
│   ├── main.server.go      # Optional
│   └── main.wasm.go        # Optional
├── spa/                    # SPA Architecture (priority 2) 
│   ├── theme/              
│   ├── public/             
│   ├── main.server.go      
│   └── main.wasm.go        
├── mpa/                    # MPA Architecture (priority 3)
│   ├── theme/              
│   ├── public/             
│   ├── main.server.go      
│   └── main.wasm.go        
└── go.mod
```

## Requirements

### 1. AppType Constants Update
- Rename `AppTypeWeb` → `AppTypeMPA` 
- Remove references to `web/` subdirectories
- Add `AppTypeMPA` as the third web architecture type

### 2. Architecture Detection Rules
- **Single Architecture**: Only ONE web architecture allowed (`pwa/` OR `spa/` OR `mpa/`)
- **Priority Order**: PWA (1) > SPA (2) > MPA (3) - highest priority wins
- **Console Compatibility**: Can coexist with any single web architecture
- **Default Behavior**: If no architecture directories exist, return `AppTypeUnknown`

### 3. AutoConfig Responsibilities
**ONLY Detection - NO Creation:**
- ✅ Detect application types (`AppType`)
- ✅ Validate single web architecture rule
- ✅ Apply priority order when conflicts exist
- ❌ Create directories
- ❌ Define required files per architecture
- ❌ Handle architecture-specific files

### 4. Architecture Handlers (External Dependencies)
Each architecture will have its own handler integrated into the central `handler`:
- `PwaHandler` - Manages PWA-specific logic
- `SpaHandler` - Manages SPA-specific logic  
- `MpaHandler` - Manages MPA-specific logic

### 5. Conflict Resolution
- **Multiple Architectures Found**: Show warning and apply priority order
- **Priority Logic**: 
  - If `pwa/` AND `spa/` exist → Use PWA, warn about SPA
  - If `spa/` AND `mpa/` exist → Use SPA, warn about MPA
  - If `pwa/` AND `mpa/` exist → Use PWA, warn about MPA

### 6. Configuration Method Updates
Update these methods to work with new architecture:
- `GetWebFilesFolder()` → Return detected architecture directory name
- `GetOutputStaticsDirectory()` → Return `{architecture}/public`
- `GetPublicFolder()` → Remain as `"public"`
- Keep existing methods: `GetServerPort()`, `GetWebServerFileName()`, `GetCMDFileName()`

### 7. Path Detection Updates
Update `isRelevantDirectoryChange()` to monitor:
- `cmd`
- `cmd/{AppName}`
- `pwa` (new)
- `spa` (new)  
- `mpa` (new)
- Remove: `web`, `web/pwa`, `web/spa`

## Implementation Steps

### Phase 1: AppType Constants
1. Update constants in `autoconfig.go`:
   - `AppTypeWeb` → `AppTypeMPA`
   - Update comments to reflect new structure

### Phase 2: Detection Logic  
1. Update `scanDirectoryStructure()`:
   - Remove `web/` directory scanning
   - Add direct root-level architecture detection
   - Implement priority order logic

2. Update `detectAllWebArchitectures()`:
   - Scan root directory instead of `web/`
   - Add MPA detection
   - Return architectures in priority order

3. Update `validateArchitecture()`:
   - Implement priority-based conflict resolution
   - Generate appropriate warnings

### Phase 3: Configuration Methods
1. Update path-related methods:
   - `GetWebFilesFolder()` → return architecture name
   - `GetOutputStaticsDirectory()` → use new paths

2. Update `isRelevantDirectoryChange()`:
   - Remove `web/*` paths
   - Add direct architecture paths

### Phase 4: Tests
1. Update all existing tests in `autoconfig_test.go`
2. Add priority conflict resolution tests:
   - Test PWA vs SPA priority
   - Test SPA vs MPA priority  
   - Test PWA vs MPA priority
   - Document expected behavior in test names

### Phase 5: Documentation
1. Update `README.md` structure examples
2. Update architecture diagrams if needed:
   - `godev-architecture.puml`
   - `godev-component-flow.puml`

## Test Cases Required

### Priority Resolution Tests
```go
// TestArchDetector_PWA_SPA_Priority - PWA wins over SPA
// TestArchDetector_SPA_MPA_Priority - SPA wins over MPA  
// TestArchDetector_PWA_MPA_Priority - PWA wins over MPA
```

### Architecture Detection Tests
```go
// TestArchDetector_DetectPWA_RootLevel - pwa/ in root
// TestArchDetector_DetectSPA_RootLevel - spa/ in root
// TestArchDetector_DetectMPA_RootLevel - mpa/ in root
// TestArchDetector_NoArchitecture_ReturnsUnknown
```

### Hybrid Application Tests
```go
// TestArchDetector_CMD_PWA_Hybrid
// TestArchDetector_CMD_SPA_Hybrid  
// TestArchDetector_CMD_MPA_Hybrid
```

## Breaking Changes
- **No Backward Compatibility**: Projects with `web/` structure will need migration
- **Path Changes**: All hardcoded `web/` paths must be updated
- **Handler Integration**: New architecture handlers need integration points

## Success Criteria
1. ✅ All tests pass with new structure
2. ✅ Priority order correctly resolves conflicts
3. ✅ `AppTypeUnknown` returned when no architecture found
4. ✅ Configuration methods work with new paths
5. ✅ No directory creation in AutoConfig
6. ✅ Documentation reflects new structure

## Notes
- Architecture handlers (PwaHandler, SpaHandler, MpaHandler) will be external dependencies
- Central `handler` manages all architecture handlers similar to current pattern
- AutoConfig only detects, does not create or manage files
- Public files like `manifest.json` and `sw.js` go in `{architecture}/public/`
