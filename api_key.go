package app

import (
    "crypto/rand"
    "encoding/hex"
    "errors"
    "os"
)

// generateAPIKey generates a cryptographically secure random 32-byte hex key.
func generateAPIKey() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}

// loadOrCreateAPIKey reads the key from path; generates and persists if absent.
// Returns ("", nil) when path is empty — caller treats this as open/no-auth mode.
func loadOrCreateAPIKey(path string) (string, error) {
    if path == "" {
        return "", nil
    }
    data, err := os.ReadFile(path)
    if err == nil {
        return string(data), nil
    }
    if !errors.Is(err, os.ErrNotExist) {
        return "", err
    }
    key, err := generateAPIKey()
    if err != nil {
        return "", err
    }
    return key, os.WriteFile(path, []byte(key), 0600)
}

// readAPIKey reads the key from path without generating.
// Used by runClient (separate process, daemon already created the key).
func readAPIKey(path string) string {
    if path == "" {
        return ""
    }
    data, _ := os.ReadFile(path)
    return string(data)
}
