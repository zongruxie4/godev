package godev

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

// GetWebFilesFolder returns the conventional web files folder path
func (c *Config) GetWebFilesFolder() string {
	return "src" // Fixed conventional path
}

// GetPublicFolder returns the public folder path
func (c *Config) GetPublicFolder() string {
	return "public" // Relative to webclient/
}

// GetOutputStaticsDirectory returns the output directory for static files
func (c *Config) GetOutputStaticsDirectory() string {
	return filepath.Join(c.GetWebFilesFolder(), "webclient", c.GetPublicFolder())
	// Returns: "src/webclient/public"
}

// GetServerPort returns the default server port
func (c *Config) GetServerPort() string {
	return "4430" // Default HTTPS development port
}

// GetRootDir returns the root directory
func (c *Config) GetRootDir() string {
	return c.rootDir
}

// GetWebServerFileName returns only the filename for web server
func (c *Config) GetWebServerFileName() string {
	return "main.server.go"
}

// GetWorkerFileName returns only the filename for edge worker
func (c *Config) GetWorkerFileName() string {
	return "main.worker.go"
}

// GetCMDFileName returns only the filename for console application
func (c *Config) GetCMDFileName() string {
	return "main.wasm.go"
}
