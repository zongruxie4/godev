package test

import (
	"github.com/tinywasm/app"
	"github.com/tinywasm/server"
)

// init sets app.TestMode=true for all tests in this package.
func init() {
	app.TestMode = true
	server.TestMode = true
}

// mockStore avoids using the real DB
type mockStore struct {
	data map[string]string
}

func newMockStore() *mockStore {
	return &mockStore{data: make(map[string]string)}
}

func (m *mockStore) Get(key string) (string, error) {
	return m.data[key], nil
}

func (m *mockStore) Set(key, value string) error {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[key] = value
	return nil
}

func (m *mockStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

// countOccurrences counts occurrences of substr in s.
func countOccurrences(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
