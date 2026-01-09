package app

import (
	"github.com/tinywasm/server"
)

// init sets TestMode=true for all tests in this package.
// This file is only included in test builds (_test.go suffix),
// ensuring these settings don't leak into production.
func init() {
	TestMode = true
	// SAFETY: Centralized static configuration for tests
	server.TestMode = true
}
