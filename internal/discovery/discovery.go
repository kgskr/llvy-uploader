// Package discovery performs bounded, metadata-only replay discovery.
package discovery

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const MaxSize int64 = 64 << 20
const MissingFolder = "리플레이 폴더를 찾지 못했습니다. LoL 클라이언트의 리플레이 저장 위치를 확인한 뒤 폴더를 선택하세요."
const DownloadRequired = "로컬 다운로드 필요: 탐색기에서 항상 이 장치에 유지를 선택한 뒤 다시 찾으세요"

type File struct {
	Path, Name string
	Size       int64
	ModTime    time.Time
	Problem    string
}
type Candidate struct{ Path, Source string }
type Result struct {
	Candidates []Candidate
	Warnings   []string
}

// Reader allows filesystem errors and Windows metadata to be tested independently.
type Reader interface {
	ReadDir(string) ([]fs.DirEntry, error)
	Inspect(string) (File, bool, error)
}
type OSReader struct{}

func (OSReader) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }

// Find honors a usable saved choice. Other candidates are bounded caller inputs;
// installation roots are accepted only when a direct replay is present.
func Find(ctx context.Context, reader Reader, saved string, candidates []Candidate) Result {
	result := Result{}
	if saved != "" {
		if _, err := reader.ReadDir(saved); err == nil {
			result.Candidates = []Candidate{{saved, "저장한 경로"}}
			return result
		}
		result.Warnings = append(result.Warnings, "저장한 리플레이 폴더에 접근할 수 없습니다")
	}
	seen := map[string]bool{}
	for _, c := range candidates {
		if ctx.Err() != nil {
			break
		}
		key := strings.ToLower(filepath.Clean(c.Path))
		if seen[key] {
			continue
		}
		seen[key] = true
		entries, err := reader.ReadDir(c.Path)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				result.Warnings = append(result.Warnings, c.Source+": 폴더 접근 오류")
			}
			continue
		}
		if c.Source == "설치 경로의 리플레이" {
			found := false
			for _, e := range entries {
				if strings.EqualFold(filepath.Ext(e.Name()), ".rofl") {
					_, include, err := reader.Inspect(filepath.Join(c.Path, e.Name()))
					if err == nil && include {
						found = true
						break
					}
				}
			}
			if !found {
				continue
			}
		}
		result.Candidates = append(result.Candidates, c)
	}
	return result
}

func Scan(ctx context.Context, reader Reader, folder string) ([]File, error) {
	entries, err := reader.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	files := make([]File, 0)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".rofl") {
			continue
		}
		path := filepath.Join(folder, entry.Name())
		file, include, err := reader.Inspect(path)
		if err != nil {
			file = File{Path: path, Name: entry.Name(), Problem: "파일 상태를 읽을 수 없습니다"}
			include = true
		}
		if !include {
			continue
		}
		if file.Problem == "" {
			if file.Size == 0 {
				file.Problem = "빈 파일"
			} else if file.Size > MaxSize {
				file.Problem = "64 MiB 초과"
			}
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}
