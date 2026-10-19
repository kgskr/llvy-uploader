package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLockAndRemoval(t *testing.T) {
	base := t.TempDir()
	s, err := OpenStore(base, "origin-a")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if other, err := OpenStore(base, "origin-a"); err == nil {
		other.Close()
		t.Fatal("second instance acquired lock")
	}
	other, err := OpenStore(base, "origin-b")
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
	for _, folder := range []string{"한글 폴더", "다른 폴더"} {
		if err := s.Save(Config{ReplayFolder: folder}); err != nil {
			t.Fatal(err)
		}
		c, err := s.Load()
		if err != nil || c.ReplayFolder != folder {
			t.Fatalf("%+v %v", c, err)
		}
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.dir, "instance.lock")); err != nil {
		t.Fatal(err)
	}
	if other, err := OpenStore(base, "origin-a"); err == nil {
		other.Close()
		t.Fatal("clear released lock")
	}
	s.Close()
	again, err := OpenStore(base, "origin-a")
	if err != nil {
		t.Fatal(err)
	}
	again.Close()
}
func TestCorruptConfig(t *testing.T) {
	s, err := OpenStore(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := os.WriteFile(filepath.Join(s.dir, "config.json"), []byte(`{"replayFolder":"x"} {}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err == nil {
		t.Fatal("trailing JSON accepted")
	}
}
