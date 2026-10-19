package app

import (
	"fmt"
	"llvy-uploader/internal/discovery"
	"llvy-uploader/internal/matching"
)

type Row struct {
	ID         matching.ID
	Candidates []discovery.File
	Selected   int
}

func Preview(ids []matching.ID, files []discovery.File) []Row {
	rows := make([]Row, 0, len(ids))
	for _, id := range ids {
		row := Row{ID: id, Selected: -1}
		for _, file := range files {
			if matching.Matches(id, file.Name) {
				row.Candidates = append(row.Candidates, file)
			}
		}
		if len(row.Candidates) == 1 && row.Candidates[0].Problem == "" {
			row.Selected = 0
		}
		rows = append(rows, row)
	}
	return rows
}
func (r Row) Status() string {
	if len(r.Candidates) == 0 {
		return "파일 없음"
	}
	if len(r.Candidates) == 1 && r.Candidates[0].Problem != "" {
		return r.Candidates[0].Problem
	}
	if r.Selected < 0 {
		return "파일 선택 필요"
	}
	return "선택됨"
}
func Select(rows []Row, row, candidate int) error {
	if row < 0 || row >= len(rows) || candidate < 0 || candidate >= len(rows[row].Candidates) {
		return fmt.Errorf("유효하지 않은 파일 선택")
	}
	if problem := rows[row].Candidates[candidate].Problem; problem != "" {
		return fmt.Errorf("%s", problem)
	}
	rows[row].Selected = candidate
	return nil
}

// Selected deduplicates by resolved path; disk identity is checked again before transfer.
func Selected(rows []Row) []discovery.File {
	seen := map[string]bool{}
	var files []discovery.File
	for _, r := range rows {
		if r.Selected < 0 || r.Selected >= len(r.Candidates) {
			continue
		}
		f := r.Candidates[r.Selected]
		if f.Problem == "" && !seen[f.Path] {
			files = append(files, f)
			seen[f.Path] = true
		}
	}
	return files
}
