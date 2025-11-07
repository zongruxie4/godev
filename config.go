package golite

import (
	"path/filepath"
)

// Config holds conventional configuration paths for Go projects
// using the standard src/ directory structure
type Config struct {
	rootDir string               // Root directory (default: ".")
	logger  func(message ...any) // Logging function
	AppName string               // Application name (directory name)
}

// NewConfig creates a new configuration with conventional paths
func NewConfig(rootDir string, logger func(message ...any)) *Config {
	root := "." // Default to current directory

	if rootDir != root {
		root = rootDir
	}

	return &Config{
		rootDir: root,
		logger:  logger,
		AppName: filepath.Base(root),
	}
}

// GetAppName returns the detected application name
func (c *Config) GetAppName() string {
	if c.AppName == "" {
		return filepath.Base(c.rootDir)
	}
	return c.AppName
}

// === BASE DIRECTORIES ===

// SrcDir returns the relative source directory path
// Returns: "src"
func (c *Config) SrcDir() string {
	return "src"
}

// CmdDir returns the relative command directory path
// Returns: "src/cmd"
func (c *Config) CmdDir() string {
	return filepath.Join(c.SrcDir(), "cmd")
}

// WebDir returns the relative web directory path
// Returns: "src/web"
func (c *Config) WebDir() string {
	return filepath.Join(c.SrcDir(), "web")
}

// DeployDir returns the relative deployment directory path
// Returns: "deploy"
func (c *Config) DeployDir() string {
	return "deploy"
}

// === CMD ENTRY POINTS ===

// CmdAppServerDir returns the relative appserver command directory path
// Returns: "src/cmd/appserver"
func (c *Config) CmdAppServerDir() string {
	return filepath.Join(c.CmdDir(), "appserver")
}

// CmdWebClientDir returns the relative webclient command directory path
// Returns: "src/cmd/webclient"
func (c *Config) CmdWebClientDir() string {
	return filepath.Join(c.CmdDir(), "webclient")
}

// CmdEdgeWorkerDir returns the relative edgeworker command directory path
// Returns: "src/cmd/edgeworker"
func (c *Config) CmdEdgeWorkerDir() string {
	return filepath.Join(c.CmdDir(), "edgeworker")
}

// === WEB DIRECTORIES ===

// WebPublicDir returns the relative web public directory path
// Returns: "src/web/public"
func (c *Config) WebPublicDir() string {
	return filepath.Join(c.WebDir(), "public")
}

// WebUIDir returns the relative web UI directory path
// Returns: "src/web/ui"
func (c *Config) WebUIDir() string {
	return filepath.Join(c.WebDir(), "ui")
}

// JsDir returns the relative web JavaScript directory path
// Returns: "src/web/ui/js"
func (c *Config) JsDir() string {
	return filepath.Join(c.WebUIDir(), "js")
}

// === DEPLOY DIRECTORIES ===

// DeployEdgeWorkerDir returns the relative edgeworker deployment directory path
// Returns: "deploy/edgeworker"
func (c *Config) DeployEdgeWorkerDir() string {
	return filepath.Join(c.DeployDir(), "edgeworker")
}

// DeployAppServerDir returns the relative appserver deployment directory path
// Returns: "deploy/appserver"
func (c *Config) DeployAppServerDir() string {
	return filepath.Join(c.DeployDir(), "appserver")
}

// === CONFIGURATION ===

// ServerPort returns the default server port
func (c *Config) ServerPort() string {
	return "6060" // Default HTTPS development port
}

// RootDir returns the root directory
func (c *Config) RootDir() string {
	return c.rootDir
}
