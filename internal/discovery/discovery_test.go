package discovery

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDirectScan(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"KR-123.ROFL", "empty.rofl", "ignore.txt"} {
		data := []byte("synthetic")
		if name == "empty.rofl" {
			data = nil
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "nested.rofl"), 0700); err != nil {
		t.Fatal(err)
	}
	files, err := Scan(context.Background(), OSReader{}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Name != "KR-123.ROFL" || files[1].Problem != "빈 파일" {
		t.Fatalf("%+v", files)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Scan(ctx, OSReader{}, dir); err != context.Canceled {
		t.Fatal(err)
	}
}
func TestCloudMetadata(t *testing.T) {
	for _, tt := range []struct {
		name              string
		attributes, tag   uint32
		include, download bool
	}{
		{"local", 0x400, 0x9000001a, true, false}, {"pinned", 0x80400, 0x9000001a, true, false},
		{"online", 0x401400, 0x9000001a, true, true}, {"partial", 0x400400, 0x9000001a, true, true},
		{"recall", 0x40400, 0x9000001a, true, true}, {"symlink", 0x400, 0xa000000c, false, false},
		{"junction", 0x400, 0xa0000003, false, false}, {"directory", 0x10, 0, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f, include, err := FromWindowsMetadata("KR-1.rofl", tt.attributes, tt.tag, 10, time.Time{})
			if err != nil || include != tt.include || (f.Problem != "") != tt.download {
				t.Fatalf("%+v %v %v", f, include, err)
			}
		})
	}
}

type fakeReader struct {
	directories map[string][]fs.DirEntry
	failures    map[string]error
}

func (f fakeReader) ReadDir(path string) ([]fs.DirEntry, error) {
	if err := f.failures[path]; err != nil {
		return nil, err
	}
	e, ok := f.directories[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return e, nil
}
func (f fakeReader) Inspect(path string) (File, bool, error) { return File{}, true, nil }
func TestDiscoveryPriorityAndErrors(t *testing.T) {
	f := fakeReader{directories: map[string][]fs.DirEntry{"saved": {}, "docs": {}, "installed": {}}, failures: map[string]error{"denied": fs.ErrPermission}}
	candidates := []Candidate{{"docs", "Windows 문서 폴더"}, {"installed", "설치 경로의 리플레이"}, {"denied", "설치 경로"}}
	r := Find(context.Background(), f, "saved", candidates)
	if len(r.Candidates) != 1 || r.Candidates[0].Path != "saved" {
		t.Fatal(r)
	}
	r = Find(context.Background(), f, "missing", candidates)
	if len(r.Candidates) != 1 || r.Candidates[0].Path != "docs" || len(r.Warnings) != 2 {
		t.Fatal(r)
	}
}
