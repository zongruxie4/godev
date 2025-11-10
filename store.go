package golite

import "os"

type FileStore struct{}

func (fs FileStore) GetFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs FileStore) SetFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// AddToFile appends bytes to the end of the named file. This is used by
// tinydb when inserting new key/value pairs to avoid rewriting the whole
// store on every insert.
func (fs FileStore) AddToFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}
