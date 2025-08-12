package godev

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// AppType represents the type of application detected
type AppType string

const (
	AppTypeUnknown AppType = "unknown"
	AppTypeConsole AppType = "console" // cmd/
	AppTypeMPA     AppType = "mpa"     // mpa/ (Multi-Page Application)
	AppTypePWA     AppType = "pwa"     // pwa/ (Progressive Web App)
	AppTypeSPA     AppType = "spa"     // spa/ (Single Page Application)
)

// AutoConfig handles automatic detection of application architecture
// based on directory structure following convention over configuration
type AutoConfig struct {
	rootDir    string    // Root directory to scan (default: ".")
	logger     io.Writer // Logging writer
	AppName    string    // Application name (directory name)
	Types      []AppType // Detected application types
	HasConsole bool      // Has cmd/ directory
	WebType    AppType   // Web architecture type (pwa, spa, or web if undefined)
}

// NewAutoConfig creates a new auto configuration detector
func NewAutoConfig(rootDir string, logger io.Writer) *AutoConfig {
	root := "." // Default to current directory

	if rootDir != root {
		root = rootDir
	}

	return &AutoConfig{
		rootDir: root,
		logger:  logger,
		AppName: filepath.Base(root),
		Types:   []AppType{},
		WebType: AppTypeUnknown,
	}
}

// Configuration methods to replace config.go functionality

// GetAppName returns the detected application name
func (ac *AutoConfig) GetAppName() string {
	if ac.AppName == "" {
		return filepath.Base(ac.rootDir)
	}
	return ac.AppName
}

// GetWebFilesFolder returns the detected web architecture folder path
func (ac *AutoConfig) GetWebFilesFolder() string {
	switch ac.WebType {
	case AppTypePWA:
		return "pwa"
	case AppTypeSPA:
		return "spa"
	case AppTypeMPA:
		return "mpa"
	default:
		return "pwa" // Default fallback
	}
}

// GetPublicFolder returns the public folder path
func (ac *AutoConfig) GetPublicFolder() string {
	return "public" // Convention: public files are in "web/public/"
}

// GetOutputStaticsDirectory returns the output directory for static files
func (ac *AutoConfig) GetOutputStaticsDirectory() string {
	return filepath.Join(ac.GetWebFilesFolder(), ac.GetPublicFolder())
}

// GetServerPort returns the default server port
func (ac *AutoConfig) GetServerPort() string {
	return "4430" // Default HTTPS development port
}

// GetRootDir returns the root directory
func (ac *AutoConfig) GetRootDir() string {
	return ac.rootDir
}

// HasWebArchitecture returns true if the project has web components
func (ac *AutoConfig) HasWebArchitecture() bool {
	return ac.WebType != AppTypeUnknown
}

// GetWebServerFileName returns only the filename for web server
func (ac *AutoConfig) GetWebServerFileName() string {
	return "main.server.go"
}

// GetCMDFileName returns only the filename for console application
func (ac *AutoConfig) GetCMDFileName() string {
	return "main.go"
}

// NewFolderEvent implements the FolderEvent interface for folder change notifications
// This is the ONLY method for architecture detection (both initial and runtime changes)
func (ac *AutoConfig) NewFolderEvent(folderName, path, event string) error {
	// Only process directory events
	if !ac.isRelevantDirectoryChange(path) {
		return nil // Not a directory we care about
	}

	fmt.Fprintln(ac.logger, fmt.Sprintf("AutoConfig: Directory %s detected (%s)", event, path))

	// Perform full architecture scan after any relevant directory change
	return ac.ScanDirectoryStructure()
}

// ScanDirectoryStructure performs a full scan and updates the architecture
func (ac *AutoConfig) ScanDirectoryStructure() error {
	// Store old state for comparison
	oldTypes := slices.Clone(ac.Types)
	oldWebType := ac.WebType
	oldHasConsole := ac.HasConsole

	// Reset current state
	ac.Types = []AppType{}
	ac.HasConsole = false
	ac.WebType = AppTypeUnknown

	// Scan directory structure
	if err := ac.scanDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to scan directory structure: %w", err)
	}

	// Validate architecture
	if err := ac.validateArchitecture(); err != nil {
		return fmt.Errorf("architecture validation failed: %w", err)
	}

	// Check if architecture changed
	if ac.hasArchitectureChanged(oldTypes, oldWebType, oldHasConsole) {
		fmt.Fprintln(ac.logger, fmt.Sprintf("AutoConfig: Architecture updated - App: %s, Types: %v", ac.AppName, ac.Types))
	}

	return nil
}

// scanDirectoryStructure scans the root directory for architecture patterns
func (ac *AutoConfig) scanDirectoryStructure() error {
	var detectedTypes []AppType

	// Check for console application (cmd/)
	cmdPath := filepath.Join(ac.rootDir, "cmd")
	if ac.directoryExists(cmdPath) {
		detectedTypes = append(detectedTypes, AppTypeConsole)
		ac.HasConsole = true
		fmt.Fprintln(ac.logger, "AutoConfig: Found console application (cmd/)")
	}

	// Check for web architectures directly in root (pwa/, spa/, mpa/)
	webTypes := ac.detectAllWebArchitectures()

	if len(webTypes) > 1 {
		// Multiple web architectures detected - apply priority order
		priorityType := ac.resolvePriorityConflict(webTypes)
		detectedTypes = append(detectedTypes, priorityType)
		ac.WebType = priorityType
	} else if len(webTypes) == 1 {
		// Single web architecture detected
		detectedTypes = append(detectedTypes, webTypes[0])
		ac.WebType = webTypes[0]
	} else {
		// No web architecture found - return unknown
		ac.WebType = AppTypeUnknown
	}

	// Log what was found
	for _, webType := range webTypes {
		switch webType {
		case AppTypePWA:
			fmt.Fprintln(ac.logger, "AutoConfig: Found Progressive Web App (pwa/)")
		case AppTypeSPA:
			fmt.Fprintln(ac.logger, "AutoConfig: Found Single Page Application (spa/)")
		case AppTypeMPA:
			fmt.Fprintln(ac.logger, "AutoConfig: Found Multi-Page Application (mpa/)")
		}
	}

	if len(webTypes) == 0 {
		fmt.Fprintln(ac.logger, "AutoConfig: No web architecture found - returning unknown")
	}

	ac.Types = detectedTypes
	return nil
}

// detectAllWebArchitectures detects all web architecture types present in root directory
func (ac *AutoConfig) detectAllWebArchitectures() []AppType {
	var webTypes []AppType

	// Check for PWA (Progressive Web App)
	pwaPath := filepath.Join(ac.rootDir, "pwa")
	if ac.directoryExists(pwaPath) {
		webTypes = append(webTypes, AppTypePWA)
	}

	// Check for SPA (Single Page Application)
	spaPath := filepath.Join(ac.rootDir, "spa")
	if ac.directoryExists(spaPath) {
		webTypes = append(webTypes, AppTypeSPA)
	}

	// Check for MPA (Multi-Page Application)
	mpaPath := filepath.Join(ac.rootDir, "mpa")
	if ac.directoryExists(mpaPath) {
		webTypes = append(webTypes, AppTypeMPA)
	}

	return webTypes
}

// detectWebArchitecture determines the specific web architecture type in root directory
func (ac *AutoConfig) detectWebArchitecture() AppType {
	// Check for PWA (Progressive Web App)
	pwaPath := filepath.Join(ac.rootDir, "pwa")
	if ac.directoryExists(pwaPath) {
		return AppTypePWA
	}

	// Check for SPA (Single Page Application)
	spaPath := filepath.Join(ac.rootDir, "spa")
	if ac.directoryExists(spaPath) {
		return AppTypeSPA
	}

	// Check for MPA (Multi-Page Application)
	mpaPath := filepath.Join(ac.rootDir, "mpa")
	if ac.directoryExists(mpaPath) {
		return AppTypeMPA
	}

	// No web architecture found
	return AppTypeUnknown
}

// validateArchitecture checks for conflicting architecture patterns
func (ac *AutoConfig) validateArchitecture() error {
	// Check for conflicting web architectures
	webCount := 0
	for _, appType := range ac.Types {
		if appType == AppTypePWA || appType == AppTypeSPA || appType == AppTypeMPA {
			webCount++
		}
	}

	if webCount > 1 {
		return errors.New("conflicting web architectures detected: only one web architecture allowed (pwa/, spa/, or mpa/)")
	}

	// Multiple console applications are not allowed
	consoleCount := 0
	for _, appType := range ac.Types {
		if appType == AppTypeConsole {
			consoleCount++
		}
	}

	if consoleCount > 1 {
		return errors.New("multiple console applications detected: only one cmd/ directory allowed")
	}

	return nil
}

// directoryExists checks if a directory exists
func (ac *AutoConfig) directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// isRelevantDirectoryChange checks if a directory change affects architecture
func (ac *AutoConfig) isRelevantDirectoryChange(dirPath string) bool {
	// Get relative path from root
	relPath, err := filepath.Rel(ac.rootDir, dirPath)
	if err != nil {
		return false
	}
	relPath = filepath.ToSlash(relPath)

	// Relevant paths that affect architecture detection
	relevantPaths := []string{
		"cmd",
		"cmd/" + ac.AppName,
		"pwa",
		"spa",
		"mpa",
	}

	for _, relevantPath := range relevantPaths {
		if relPath == relevantPath || strings.HasPrefix(relPath, relevantPath+"/") {
			return true
		}
	}

	return false
}

// hasArchitectureChanged compares old and new architecture state to detect changes
func (ac *AutoConfig) hasArchitectureChanged(oldTypes []AppType, oldWebType AppType, oldHasConsole bool) bool {
	// Check if Types slice changed
	if len(ac.Types) != len(oldTypes) {
		return true
	}

	// Check if any type changed
	typeMap := make(map[AppType]bool)
	for _, t := range oldTypes {
		typeMap[t] = true
	}

	for _, t := range ac.Types {
		if !typeMap[t] {
			return true
		}
	}

	// Check if WebType or HasConsole changed
	return ac.WebType != oldWebType || ac.HasConsole != oldHasConsole
}

// resolvePriorityConflict applies priority order when multiple web architectures are found
// Priority: PWA (1) > SPA (2) > MPA (3) - highest priority wins
func (ac *AutoConfig) resolvePriorityConflict(webTypes []AppType) AppType {
	var conflicts []string

	// Check which architectures are present and build conflict message
	hasPWA := slices.Contains(webTypes, AppTypePWA)
	hasSPA := slices.Contains(webTypes, AppTypeSPA)
	hasMPA := slices.Contains(webTypes, AppTypeMPA)

	if hasPWA {
		conflicts = append(conflicts, "pwa/")
	}
	if hasSPA {
		conflicts = append(conflicts, "spa/")
	}
	if hasMPA {
		conflicts = append(conflicts, "mpa/")
	}

	// Apply priority order and warn about conflicts
	if hasPWA {
		fmt.Fprintln(ac.logger, fmt.Sprintf("AutoConfig: Warning - Multiple web architectures found: %v. Using PWA (highest priority)", conflicts))
		return AppTypePWA
	} else if hasSPA {
		fmt.Fprintln(ac.logger, fmt.Sprintf("AutoConfig: Warning - Multiple web architectures found: %v. Using SPA (priority 2)", conflicts))
		return AppTypeSPA
	} else if hasMPA {
		fmt.Fprintln(ac.logger, fmt.Sprintf("AutoConfig: Warning - Multiple web architectures found: %v. Using MPA (priority 3)", conflicts))
		return AppTypeMPA
	}

	// Fallback (should never reach here)
	return AppTypeUnknown
}
