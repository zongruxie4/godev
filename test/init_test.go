package test

import (
	"github.com/tinywasm/app"
	"github.com/tinywasm/server"
)

// init sets app.TestMode=true for all tests in this package.
// This file is only included in test builds (_test.go suffix),
// ensuring these settings don't leak into production.
func init() {
	app.TestMode = true
	// SAFETY: Centralized static configuration for tests
	server.TestMode = true
}
