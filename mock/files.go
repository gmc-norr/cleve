package mock

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func WriteTempFile(t *testing.T, filename string, content string) (string, error) {
	d := t.TempDir()
	path := filepath.Join(d, filename)
	f, err := os.Create(path)
	if err != nil {
		return path, err
	}
	defer f.Close()
	if _, err := io.WriteString(f, content); err != nil {
		return path, err
	}
	return path, nil
}
