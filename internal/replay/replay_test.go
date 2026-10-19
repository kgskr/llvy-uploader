package replay

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"llvy-uploader/internal/discovery"
	"os"
	"path/filepath"
	"testing"
)

func TestStableHashAndRewind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "KR-123.rofl")
	data := []byte("synthetic replay bytes")
	if err := os.WriteFile(path, data, 0600); err != nil {
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
	hash, err := h.Hash(context.Background())
	if err != nil || hash != fmt.Sprintf("%x", sha256.Sum256(data)) {
		t.Fatalf("%s %v", hash, err)
	}
	got, err := io.ReadAll(h)
	if err != nil || string(got) != string(data) {
		t.Fatal("hash did not rewind")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.Hash(ctx); err != context.Canceled {
		t.Fatal(err)
	}
}
func TestChangedBeforeOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "123.rofl")
	if err := os.WriteFile(path, []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := discovery.Scan(context.Background(), discovery.OSReader{}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if h, err := Open(files[0]); err == nil {
		h.Close()
		t.Fatal("changed file accepted")
	}
}
