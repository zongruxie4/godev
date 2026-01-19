package app

import (
	"path/filepath"
)

// Config holds conventional configuration paths for Go projects
// using the root directory as source
type Config struct {
	RootDir string               // Root directory (default: ".")
	logger  func(message ...any) // Logging function
	AppName string               // Application name (directory name)
}

// NewConfig creates a new configuration with conventional paths
func NewConfig(RootDir string, logger func(message ...any)) *Config {
	root := "." // Default to current directory

	if RootDir != root {
		root = RootDir
	}

	return &Config{
		RootDir: root,
		logger:  logger,
		AppName: filepath.Base(root),
	}
}

// Name returns the Handler name for Loggable interface
func (c *Config) Name() string {
	return "Config"
}

// SetLog implements Loggable interface
func (c *Config) SetLog(f func(message ...any)) {
	c.logger = f
}

// GetAppName returns the detected application name
func (c *Config) GetAppName() string {
	if c.AppName == "" {
		return filepath.Base(c.RootDir)
	}
	return c.AppName
}

// === BASE DIRECTORIES ===

// SrcDir returns the relative source directory path
// Returns: "."
func (c *Config) SrcDir() string {
	return "."
}

// CmdDir returns the relative command directory path
// Returns: "cmd"
func (c *Config) CmdDir() string {
	return filepath.Join(c.SrcDir(), "cmd")
}

// WebDir returns the relative web directory path
// Returns: "web"
func (c *Config) WebDir() string {
	return filepath.Join(c.SrcDir(), "web")
}

// DeployDir returns the relative deployment directory path
// Returns: "deploy"
func (c *Config) DeployDir() string {
	return "deploy"
}

// === CMD ENTRY POINTS (Source Directories) ===

// CmdAppServerDir returns the relative appserver source directory path
// Returns: "web" (with new structure using build tags)
func (c *Config) CmdAppServerDir() string {
	return c.WebDir()
}

// CmdWebClientDir returns the relative webclient source directory path
// Returns: "web" (with new structure using build tags)
func (c *Config) CmdWebClientDir() string {
	return c.WebDir()
}

// CmdEdgeWorkerDir returns the relative edgeworker source directory path
// Returns: "cmd/edgeworker" (edge workers remain separate)
func (c *Config) CmdEdgeWorkerDir() string {
	return filepath.Join(c.CmdDir(), "edgeworker")
}

// === SOURCE FILE NAMES (Convention Defaults) ===

// ServerFileName returns the default server entry file name
// Returns: "server.go" (convention with //go:build !wasm)
func (c *Config) ServerFileName() string {
	return "server.go"
}

// ClientFileName returns the default WASM client entry file name
// Returns: "client.go" (convention with //go:build wasm)
func (c *Config) ClientFileName() string {
	return "client.go"
}

// === WEB DIRECTORIES ===

// WebPublicDir returns the relative web public directory path
// Returns: "web/public"
func (c *Config) WebPublicDir() string {
	return filepath.Join(c.WebDir(), "public")
}

// WebUIDir returns the relative web UI directory path
// Returns: "web/ui"
func (c *Config) WebUIDir() string {
	return filepath.Join(c.WebDir(), "ui")
}

// JsDir returns the relative web JavaScript directory path
// Returns: "web/ui/js"
func (c *Config) JsDir() string {
	return filepath.Join(c.WebUIDir(), "js")
}

// === DEPLOY DIRECTORIES ===

// DeployAppServerDir returns the relative appserver deployment directory path
// Returns: "web" (server compiles in same directory as source)
func (c *Config) DeployAppServerDir() string {
	return c.WebDir()
}

// DeployEdgeWorkerDir returns the relative edgeworker deployment directory path
// Returns: "deploy/edgeworker"
func (c *Config) DeployEdgeWorkerDir() string {
	return filepath.Join(c.DeployDir(), "edgeworker")
}

// === CONFIGURATION ===

// ServerPort returns the default server port
func (c *Config) ServerPort() string {
	return "6060" // Default HTTPS development port
}

// SetRootDir updates the root directory path
func (c *Config) SetRootDir(path string) {
	c.RootDir = path
}

// SetAppName updates the application name
func (c *Config) SetAppName(name string) {
	c.AppName = name
}
