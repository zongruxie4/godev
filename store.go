package app

import (
	"os"
	"sync"
)

type FileStore struct {
	shouldWrite   func() bool
	onFileCreated func(path string)
}

func (fs *FileStore) SetShouldWrite(f func() bool) {
	fs.shouldWrite = f
}

func (fs *FileStore) SetOnFileCreated(f func(path string)) {
	fs.onFileCreated = f
}

func (fs FileStore) GetFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs FileStore) SetFile(path string, data []byte) error {
	if fs.shouldWrite != nil && !fs.shouldWrite() {
		return nil
	}

	// Check if file exists before writing
	_, errStat := os.Stat(path)
	isNew := os.IsNotExist(errStat)

	err := os.WriteFile(path, data, 0644)

	// Notify if file was newly created
	if err == nil && isNew && fs.onFileCreated != nil {
		fs.onFileCreated(path)
	}

	return err
}

// AddToFile appends bytes to the end of the named file. This is used by
// tinydb when inserting new key/value pairs to avoid rewriting the whole
// store on every insert.
func (fs FileStore) AddToFile(path string, data []byte) error {
	if fs.shouldWrite != nil && !fs.shouldWrite() {
		return nil
	}

	// Check if file exists before writing
	_, errStat := os.Stat(path)
	isNew := os.IsNotExist(errStat)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)

	// Notify if file was newly created
	if err == nil && isNew && fs.onFileCreated != nil {
		fs.onFileCreated(path)
	}

	return err
}

// MemoryStore implements the kvdb.Store interface in-memory.
// It is used during tests to avoid disk I/O and side effects.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string][]byte),
	}
}

func (m *MemoryStore) GetFile(path string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, ok := m.data[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (m *MemoryStore) SetFile(path string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[path] = data
	return nil
}

func (m *MemoryStore) AddToFile(path string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[path] = append(m.data[path], data...)
	return nil
}
