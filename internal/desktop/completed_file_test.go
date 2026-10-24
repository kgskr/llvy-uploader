package desktop

import (
	"llvy-uploader/internal/discovery"
	"strings"
	"testing"
	"time"
)

func TestSuccessfulReselectionSeparatesChangedFileFromOldResult(t *testing.T) {
	for _, state := range []string{"completed", "duplicate", "pending", "processing", "failed", "outcome_unknown", ""} {
		t.Run(state, func(t *testing.T) {
			file := discovery.File{Path: "replay.rofl", Name: "replay.rofl", Size: 32, ModTime: time.Now()}
			old := NewEntry(file)
			old.ReservationAttempted = true
			old.DeviceSessionID = fixtureID
			old.Result = Result{State: state, GamePath: "/games/" + fixtureID}
			entries := map[string]*Entry{file.Path: old}
			if SelectEntry(entries, file) != old {
				t.Fatal("unchanged file lost reservation")
			}
			for _, field := range []string{"name", "size", "mtime", "problem"} {
				current := file
				switch field {
				case "name":
					current.Name = "other.rofl"
				case "size":
					current.Size++
				case "mtime":
					current.ModTime = current.ModTime.Add(time.Second)
				case "problem":
					current.Problem = "로컬 다운로드 필요"
				}
				entries[file.Path] = old
				next := SelectEntry(entries, current)
				if state == "completed" || state == "duplicate" {
					if next == old || next.Key == old.Key || !next.NeedsReservation() || next.Result.GamePath != "" || next.File != current {
						t.Fatalf("%s retained old successful upload: %+v", field, next)
					}
				} else if next != old || next.File != file {
					t.Fatalf("%s replaced recoverable reservation", field)
				}
			}
		})
	}
}

func TestAPIErrorDisplaysRetryAfterWithoutRoundingDown(t *testing.T) {
	for _, tc := range []struct {
		delay time.Duration
		want  string
	}{
		{time.Millisecond, "최소 1초"}, {30 * time.Second, "최소 30초"},
		{30*time.Second + time.Millisecond, "최소 31초"},
	} {
		for _, code := range []string{"quota_exhausted", "rate_limited"} {
			err := &APIError{Status: 429, Code: code, RetryAfter: tc.delay}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("missing retry information: %s", err)
			}
		}
	}
	if strings.Contains((&APIError{Status: 429, Code: "rate_limited"}).Error(), "최소") {
		t.Fatal("invented retry delay")
	}
}
