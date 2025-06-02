package godev

import (
	"errors"
	"fmt"
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
	AppTypeWeb     AppType = "web"     // web/ (undefined architecture)
	AppTypePWA     AppType = "pwa"     // web/pwa/
	AppTypeSPA     AppType = "spa"     // web/spa/
)

// AutoConfig handles automatic detection of application architecture
// based on directory structure following convention over configuration
type AutoConfig struct {
	rootDir    string                // Root directory to scan (default: ".")
	print      func(messages ...any) // Logging function
	AppName    string                // Application name (directory name)
	Types      []AppType             // Detected application types
	HasConsole bool                  // Has cmd/ directory
	WebType    AppType               // Web architecture type (pwa, spa, or web if undefined)
}

// NewAutoConfig creates a new auto configuration detector
func NewAutoConfig(print func(messages ...any)) *AutoConfig {
	rootDir := "." // Default to current directory
	return &AutoConfig{
		rootDir: rootDir,
		print:   print,
		AppName: filepath.Base(rootDir),
		Types:   []AppType{},
		WebType: AppTypeUnknown,
	}
}

// SetRootDir sets the root directory for testing purposes
func (ac *AutoConfig) SetRootDir(rootDir string) {
	ac.rootDir = rootDir
	ac.AppName = filepath.Base(rootDir)
}

// Configuration methods to replace config.go functionality

// GetAppName returns the detected application name
func (ac *AutoConfig) GetAppName() string {
	if ac.AppName == "" {
		return filepath.Base(ac.rootDir)
	}
	return ac.AppName
}

// GetWebFilesFolder returns the web files folder path
func (ac *AutoConfig) GetWebFilesFolder() string {
	return "web" // Convention: web files are in "web/" directory
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

	ac.print(fmt.Sprintf("AutoConfig: Directory %s detected (%s)", event, path))

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
		ac.print(fmt.Sprintf("AutoConfig: Architecture updated - App: %s, Types: %v", ac.AppName, ac.Types))
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
		ac.print("AutoConfig: Found console application (cmd/)")
	}
	// Check for web application (web/)
	webPath := filepath.Join(ac.rootDir, "web")
	if ac.directoryExists(webPath) {
		// Check for all web architecture types and detect conflicts
		webTypes := ac.detectAllWebArchitectures(webPath)

		if len(webTypes) > 1 {
			// Multiple web architectures detected - this is a conflict
			detectedTypes = append(detectedTypes, webTypes...)
		} else if len(webTypes) == 1 {
			// Single web architecture detected
			detectedTypes = append(detectedTypes, webTypes[0])
			ac.WebType = webTypes[0]
		} else {
			// Generic web application (architecture undefined)
			detectedTypes = append(detectedTypes, AppTypeWeb)
			ac.WebType = AppTypeWeb
		}

		// Log what was found
		for _, webType := range webTypes {
			switch webType {
			case AppTypePWA:
				ac.print("AutoConfig: Found Progressive Web App (web/pwa/)")
			case AppTypeSPA:
				ac.print("AutoConfig: Found Single Page Application (web/spa/)")
			}
		}

		if len(webTypes) == 0 {
			ac.print("AutoConfig: Found web project (web/) - architecture undefined")
		}
	}

	ac.Types = detectedTypes
	return nil
}

// detectAllWebArchitectures detects all web architecture types present
func (ac *AutoConfig) detectAllWebArchitectures(webPath string) []AppType {
	var webTypes []AppType

	// Check for PWA (Progressive Web App)
	pwaPath := filepath.Join(webPath, "pwa")
	if ac.directoryExists(pwaPath) {
		webTypes = append(webTypes, AppTypePWA)
	}

	// Check for SPA (Single Page Application)
	spaPath := filepath.Join(webPath, "spa")
	if ac.directoryExists(spaPath) {
		webTypes = append(webTypes, AppTypeSPA)
	}

	return webTypes
}

// detectWebArchitecture determines the specific web architecture type
func (ac *AutoConfig) detectWebArchitecture(webPath string) AppType {
	// Check for PWA (Progressive Web App)
	pwaPath := filepath.Join(webPath, "pwa")
	if ac.directoryExists(pwaPath) {
		return AppTypePWA
	}

	// Check for SPA (Single Page Application)
	spaPath := filepath.Join(webPath, "spa")
	if ac.directoryExists(spaPath) {
		return AppTypeSPA
	}

	// Generic web application (architecture undefined)
	return AppTypeWeb
}

// validateArchitecture checks for conflicting architecture patterns
func (ac *AutoConfig) validateArchitecture() error {
	// Check for conflicting web architectures
	webCount := 0
	for _, appType := range ac.Types {
		if appType == AppTypePWA || appType == AppTypeSPA {
			webCount++
		}
	}

	if webCount > 1 {
		return errors.New("conflicting web architectures detected: cannot have both PWA and SPA")
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
		"web",
		"web/pwa",
		"web/spa",
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
