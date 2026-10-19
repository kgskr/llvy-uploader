package replay

import (
	"context"
	"llvy-uploader/internal/discovery"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsReadHandleBlocksMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "한글-123.rofl")
	if err := os.WriteFile(path, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := discovery.Scan(context.Background(), discovery.OSReader{}, filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	h, err := Open(files[0])
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if writer, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
		writer.Close()
		t.Fatal("writer opened locked file")
	}
	if err := os.Remove(path); err == nil {
		t.Fatal("deleted locked file")
	}
}
