package godev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewConfig tests the constructor
func TestNewConfig(t *testing.T) {
	var logs []string
	logger := func(message ...any) {
		if len(message) > 0 {
			if str, ok := message[0].(string); ok {
				logs = append(logs, str)
			}
		}
	}
	config := NewConfig(".", logger)

	assert.NotNil(t, config)
	assert.Equal(t, ".", config.rootDir)
	assert.NotNil(t, config.logger)
	assert.NotEmpty(t, config.AppName)
}

// TestConfigSetRootDir tests setting root directory
func TestConfigSetRootDir(t *testing.T) {
	logger := func(message ...any) {
		// Test logger - do nothing
	}
	testDir := "/test/project"
	config := NewConfig(testDir, logger)

	assert.Equal(t, testDir, config.rootDir)
	assert.Equal(t, "project", config.AppName)
}

// TestGetWebFilesFolder tests that it returns fixed "src" path
func TestGetWebFilesFolder(t *testing.T) {
	config := NewConfig(".", func(message ...any) {})
	assert.Equal(t, "src", config.GetWebFilesFolder())
}

// TestGetPublicFolder tests that it returns fixed "public" path
func TestGetPublicFolder(t *testing.T) {
	config := NewConfig(".", func(message ...any) {})
	assert.Equal(t, "public", config.GetPublicFolder())
}

// TestGetOutputStaticsDirectory tests the combined path
func TestGetOutputStaticsDirectory(t *testing.T) {
	config := NewConfig(".", func(message ...any) {})
	expected := "src/webclient/public"
	assert.Equal(t, expected, config.GetOutputStaticsDirectory())
}

// TestGetServerPort tests the default server port
func TestGetServerPort(t *testing.T) {
	config := NewConfig(".", func(message ...any) {})
	assert.Equal(t, "4430", config.GetServerPort())
}

// TestGetWebServerFileName tests the web server filename
func TestGetWebServerFileName(t *testing.T) {
	config := NewConfig(".", func(message ...any) {})
	assert.Equal(t, "main.server.go", config.GetWebServerFileName())
}

// TestGetWorkerFileName tests the edge worker filename
func TestGetWorkerFileName(t *testing.T) {
	config := NewConfig(".", func(message ...any) {})
	assert.Equal(t, "main.worker.go", config.GetWorkerFileName())
}

// TestGetCMDFileName tests the console app filename
func TestGetCMDFileName(t *testing.T) {
	config := NewConfig(".", func(message ...any) {})
	assert.Equal(t, "main.wasm.go", config.GetCMDFileName())
}
