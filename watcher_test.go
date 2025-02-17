package godev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContain(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		setup    func() *WatchHandler
		expected bool
	}{
		{
			name: "hidden file",
			path: ".gitignore",
			setup: func() *WatchHandler {
				return &WatchHandler{
					WatchConfig: &WatchConfig{
						UnobservedFiles: func() []string {
							return []string{}
						},
					},
				}
			},
			expected: true,
		},
		{
			name: "unobserved file",
			path: "test/.git",
			setup: func() *WatchHandler {
				return &WatchHandler{
					WatchConfig: &WatchConfig{
						UnobservedFiles: func() []string {
							return []string{".git"}
						},
					},
				}
			},
			expected: true,
		},
		{
			name: "observed file",
			path: "test/main.go",
			setup: func() *WatchHandler {
				return &WatchHandler{
					WatchConfig: &WatchConfig{
						UnobservedFiles: func() []string {
							return []string{".git"}
						},
					},
				}
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setup()
			assert.Equal(t, tt.expected, handler.Contain(tt.path))
		})
	}
}
