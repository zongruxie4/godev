# Config Restructuring Plan - Remove Auto-Detection Logic

## 📋 Overview

**Objective:** Simplify `AutoConfig` → `Config` by removing PWA/SPA/MPA auto-detection logic and using fixed conventional paths based on the new documented architecture structure (`src/` directory).

**Current State:** `AutoConfig` dynamically scans for `pwa/`, `spa/`, `mpa/` directories and auto-detects architecture types.

**Target State:** `Config` with fixed conventional paths: `src/` as base, with subdirectories `webclient/`, `appserver/`, `edgeworker/`, `common/`, `modules/`.

---

## 🎯 Changes Required

### 1. **Rename and Simplify Type**

#### Current (`autoconfig.go`):
```go
type AutoConfig struct {
    rootDir    string
    logger     func(message ...any)
    AppName    string
    Types      []AppType     // ❌ Remove - no longer detecting types
    HasConsole bool          // ❌ Remove - not needed
    WebType    AppType       // ❌ Remove - no dynamic detection
}
```

#### New (`config.go`):
```go
type Config struct {
    rootDir string
    logger  func(message ...any)
    AppName string
}
```

**Rationale:** We no longer need to track detected types or architecture state since paths are now conventional.

---

### 2. **Remove AppType Enum and Constants**

#### Delete from `autoconfig.go`:
```go
// ❌ REMOVE ENTIRE SECTION
type AppType string

const (
    AppTypeUnknown AppType = "unknown"
    AppTypeConsole AppType = "console"
    AppTypeMPA     AppType = "mpa"
    AppTypePWA     AppType = "pwa"
    AppTypeSPA     AppType = "spa"
)
```

**Rationale:** No architecture detection = no need for type enums.

---

### 3. **Simplify Constructor**

#### Current:
```go
func NewAutoConfig(rootDir string, logger func(message ...any)) *AutoConfig {
    // ...initialization
    return &AutoConfig{
        rootDir: root,
        logger:  logger,
        AppName: filepath.Base(root),
        Types:   []AppType{},
        WebType: AppTypeUnknown,
    }
}
```

#### New:
```go
func NewConfig(rootDir string, logger func(message ...any)) *Config {
    root := "."
    if rootDir != root {
        root = rootDir
    }
    
    return &Config{
        rootDir: root,
        logger:  logger,
        AppName: filepath.Base(root),
    }
}
```

**Rationale:** Much simpler initialization without type detection fields.

---

### 4. **Update Configuration Methods**

#### Keep (with modifications):

```go
// GetAppName - NO CHANGES
func (c *Config) GetAppName() string {
    if c.AppName == "" {
        return filepath.Base(c.rootDir)
    }
    return c.AppName
}

// GetWebFilesFolder - SIMPLIFIED (always returns "src")
func (c *Config) GetWebFilesFolder() string {
    return "src"  // ✅ Fixed conventional path
}

// GetPublicFolder - NO CHANGES (already correct)
func (c *Config) GetPublicFolder() string {
    return "public"  // Relative to webclient/
}

// GetOutputStaticsDirectory - UPDATED
func (c *Config) GetOutputStaticsDirectory() string {
    return filepath.Join("src", "webclient", c.GetPublicFolder())
    // Returns: "src/webclient/public"
}

// GetServerPort - NO CHANGES
func (c *Config) GetServerPort() string {
    return "4430"
}

// GetRootDir - NO CHANGES
func (c *Config) GetRootDir() string {
    return c.rootDir
}

// GetWebServerFileName - NO CHANGES
func (c *Config) GetWebServerFileName() string {
    return "main.server.go"
}

// GetCMDFileName - NO CHANGES
func (c *Config) GetCMDFileName() string {
    return "main.go"
}
```

---

### 5. **Remove Unused Methods**

#### Delete entirely:
```go
// ❌ REMOVE - No longer needed
func (ac *AutoConfig) HasWebArchitecture() bool
func (ac *AutoConfig) NewFolderEvent(folderName, path, event string) error
func (ac *AutoConfig) ScanDirectoryStructure() error
func (ac *AutoConfig) scanDirectoryStructure() error
func (ac *AutoConfig) detectAllWebArchitectures() []AppType
func (ac *AutoConfig) detectWebArchitecture() AppType
func (ac *AutoConfig) validateArchitecture() error
func (ac *AutoConfig) directoryExists(path string) bool
func (ac *AutoConfig) isRelevantDirectoryChange(dirPath string) bool
func (ac *AutoConfig) hasArchitectureChanged(...) bool
func (ac *AutoConfig) resolvePriorityConflict(webTypes []AppType) AppType
```

**Rationale:** All scanning and detection logic is obsolete with fixed paths.

---

## 🔄 Update Usages

### Files to Update:

#### 1. **`start.go`**

**Current:**
```go
type handler struct {
    config *AutoConfig  // ❌ Change type
}
```

**New:**
```go
type handler struct {
    config *Config  // ✅ Updated type
}
```

---

#### 2. **`section-build.go`**

**Current:**
```go
// CONFIG
h.config = NewAutoConfig(h.rootDir, configLogger)
// Scan initial architecture - this must happen before AddSectionBUILD
h.config.ScanDirectoryStructure()  // ❌ REMOVE THIS LINE
```

**New:**
```go
// CONFIG
h.config = NewConfig(h.rootDir, configLogger)
// ✅ No scanning needed - using conventional paths
```

**Update paths calculations:**
```go
// SERVER - Update RootFolder calculation
RootFolder: filepath.Join(h.rootDir, h.config.GetWebFilesFolder(), "appserver"),
// Before: pwa/ or spa/ or mpa/
// After:  src/appserver/

// WASM - Update paths
WebFilesRootRelative: filepath.Join(h.config.GetWebFilesFolder(), "webclient"),
// Before: pwa/ or spa/ or mpa/
// After:  src/webclient/

WebFilesSubRelative: h.config.GetPublicFolder(),
// Stays: public/

// ASSETS - Update theme folder
ThemeFolder: func() string {
    return path.Join(h.rootDir, h.config.GetWebFilesFolder(), "web", "theme")
},
// Before: pwa/theme or spa/theme or mpa/theme
// After:  src/web/theme
```

---

#### 3. **`section-deploy.go`**

**Current:**
```go
RelativeInputDirectory:  h.config.GetWebFilesFolder(),
RelativeOutputDirectory: path.Join(h.config.GetWebFilesFolder(), "deploy"),
```

**Update to:**
```go
RelativeInputDirectory:  filepath.Join(h.config.GetWebFilesFolder(), "edgeworker"),
// Now: src/edgeworker/

RelativeOutputDirectory: path.Join(h.config.GetWebFilesFolder(), "edgeworker", "deploy"),
// Now: src/edgeworker/deploy/
```

---

#### 4. **`section-build.go` (Watcher)**

**Current:**
```go
FolderEvents: h.config,  // ❌ Remove - Config no longer implements FolderEvent interface
```

**New:**
```go
FolderEvents: nil,  // ✅ No dynamic folder event handling needed
```

**Rationale:** Without architecture detection, we don't need to watch for directory structure changes.

---

## 🧪 Tests to Update/Remove

### File: `autoconfig_test.go`

**Tests to DELETE entirely:**
```go
// ❌ DELETE - No longer scanning
- TestScanDirectoryStructure_PWA
- TestScanDirectoryStructure_SPA
- TestScanDirectoryStructure_MPA
- TestScanDirectoryStructure_ConsoleOnly
- TestScanDirectoryStructure_MultipleArchitectures
- TestValidateArchitecture_ConflictingWeb
- TestValidateArchitecture_MultipleConsole
- TestDetectAllWebArchitectures
- TestDetectWebArchitecture
- TestNewFolderEvent
- TestIsRelevantDirectoryChange
- TestHasArchitectureChanged
- TestResolvePriorityConflict
```

**Tests to UPDATE:**
```go
// ✅ UPDATE - Simplify to test only fixed paths
- TestNewAutoConfig → TestNewConfig
- TestGetWebFilesFolder (should always return "src")
- TestGetPublicFolder (should always return "public")
- TestGetOutputStaticsDirectory (should return "src/webclient/public")
```

### Integration Tests to Update:

#### `start_real_scenario_test.go`
**Current structure:**
```go
files := map[string]string{
    "modules/users/newfile.js":       "console.log('H2');",
    "modules/medical/file1.js":       "console.log('one1');",
    "pwa/theme/main.js":              "console.log(\"Hello, PWA! 2\");",
}
mainJsPath := filepath.Join(tmp, "pwa", "public", "main.js")
```

**New structure:**
```go
files := map[string]string{
    "src/modules/users/newfile.js":     "console.log('H2');",
    "src/modules/medical/file1.js":     "console.log('one1');",
    "src/web/theme/main.js":            "console.log(\"Hello, PWA! 2\");",
}
mainJsPath := filepath.Join(tmp, "src", "webclient", "public", "main.js")
```

#### `start_integration_test.go` (TestStartJSEventFlow)
**Current:**
```go
file3Path := filepath.Join(tmp, "pwa", "theme", "theme.js")
```

**New:**
```go
file3Path := filepath.Join(tmp, "src", "web", "theme", "theme.js")
```

#### `start_assetmin_test.go` (TestStartAssetMinEventFlow)
**Current:**
```go
file3Path := filepath.Join(tmp, "pwa", "theme", "theme.js")
```

**New:**
```go
file3Path := filepath.Join(tmp, "src", "web", "theme", "theme.js")
```

#### `start_delete_test.go` (TestStartDeleteFileScenario)
**Current:**
```go
mainJsPath := filepath.Join(tmp, "pwa", "public", "main.js")
```

**New:**
```go
mainJsPath := filepath.Join(tmp, "src", "webclient", "public", "main.js")
```

**New simplified tests:**
```go
func TestNewConfig(t *testing.T) {
    config := NewConfig(".", func(messages ...any) {})
    
    if config.GetWebFilesFolder() != "src" {
        t.Error("Expected GetWebFilesFolder to return 'src'")
    }
    
    if config.GetPublicFolder() != "public" {
        t.Error("Expected GetPublicFolder to return 'public'")
    }
    
    expected := "src/webclient/public"
    if config.GetOutputStaticsDirectory() != expected {
        t.Errorf("Expected %s, got %s", expected, config.GetOutputStaticsDirectory())
    }
}
```

---

## 📁 File Structure Impact

### Before:
```
godev/
├── autoconfig.go       # ~370 lines (heavy logic)
├── autoconfig_test.go  # ~450 lines (many tests)
├── start.go
├── section-build.go
└── section-deploy.go
```

### After:
```
godev/
├── config.go           # ~80 lines (simple config)
├── config_test.go      # ~50 lines (basic tests)
├── start.go
├── section-build.go
└── section-deploy.go
```

**Estimated reduction:** ~650 lines of code removed

---

## 🚀 Implementation Steps

1. **Create `config.go`** - New simplified file
2. **Update `start.go`** - Change type reference
3. **Update `section-build.go`** - Remove scan call, update paths
4. **Update `section-deploy.go`** - Update paths for edgeworker
5. **Create `config_test.go`** - Simple tests for fixed paths
6. **Delete `autoconfig.go`** - Remove old file
7. **Delete `autoconfig_test.go`** - Remove old tests
8. **Run tests** - Verify everything works

---

## ⚠️ Breaking Changes

### For Users:
- **OLD:** Projects could have `pwa/`, `spa/`, or `mpa/` directories
- **NEW:** Projects MUST follow `src/` structure with `webclient/`, `appserver/`, `edgeworker/`

### Migration Guide for Users:
```bash
# If you have a PWA structure:
pwa/           → src/
  webclient/   → src/webclient/
  public/      → src/webclient/public/
  server/      → src/appserver/
  worker/      → src/edgeworker/

# If you have SPA or MPA:
# Same migration path - consolidate to src/
```

---

## 🎯 Benefits

1. **Simplicity:** ~650 lines removed, much easier to understand
2. **Performance:** No runtime directory scanning
3. **Predictability:** Fixed conventional paths = no surprises
4. **Maintainability:** Less code = fewer bugs
5. **Documentation:** Structure matches README.md example perfectly

---

## 📊 Risk Assessment

### Low Risk:
- ✅ Internal refactoring only
- ✅ Well-defined conventional structure
- ✅ Tests will catch integration issues

### Medium Risk:
- ⚠️ Users with existing PWA/SPA/MPA projects need migration
- ⚠️ External tools depending on old structure

### Mitigation:
- 📝 Clear migration guide in release notes
- 🔄 Version bump (breaking change)
- 📚 Update all examples and documentation

---

## ✅ Validation Checklist

- [x] `config.go` created with simplified logic
- [x] `config_test.go` created with basic tests
- [x] `start.go` updated (type reference)
- [x] `section-build.go` updated (removed scan, updated paths)
- [x] `section-deploy.go` updated (edgeworker paths)
- [x] `autoconfig.go` deleted
- [x] `autoconfig_test.go` deleted
- [x] `start_real_scenario_test.go` updated (pwa/ → src/ paths)
- [x] `start_integration_test.go` updated (pwa/theme/ → src/webclient/ui/)
- [x] `start_assetmin_test.go` updated (pwa/theme/ → src/webclient/ui/)
- [x] `start_delete_test.go` updated (pwa/public/ → src/webclient/public/)
- [x] All tests pass
- [x] Example project works with new structure
- [x] Documentation updated (README.md already correct)
- [x] Release notes prepared with migration guide

---

## 📝 Notes

- This restructuring aligns godev with the documented architecture in `example/README.md`
- The new `Config` is a simple configuration holder, not a detector
- Convention over configuration: `src/` is the expected structure
- No more magic - explicit and predictable behavior
