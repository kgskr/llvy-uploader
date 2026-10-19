package app

import (
	"llvy-uploader/internal/discovery"
	"llvy-uploader/internal/matching"
	"testing"
)

func TestAmbiguityAndDedup(t *testing.T) {
	ids, _ := matching.Parse("123 KR-123 456 789")
	files := []discovery.File{{Name: "KR-123.rofl", Path: "a"}, {Name: "NA1-123.rofl", Path: "b"}, {Name: "456.rofl", Path: "c", Problem: discovery.DownloadRequired}}
	rows := Preview(ids, files)
	if rows[0].Status() != "파일 선택 필요" || rows[2].Status() != discovery.DownloadRequired || rows[3].Status() != "파일 없음" {
		t.Fatal(rows)
	}
	if err := Select(rows, 0, 0); err != nil {
		t.Fatal(err)
	}
	if len(Selected(rows)) != 1 {
		t.Fatal("same selected file duplicated")
	}
	if err := Select(rows, 2, 0); err == nil {
		t.Fatal("unavailable file selected")
	}
}
