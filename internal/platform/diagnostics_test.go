package platform

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDiagnosticsRejectArbitraryData(t *testing.T) {
	s, err := OpenStore(t.TempDir(), "test-origin")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	valid := Diagnostic{Event: EventUpload, Outcome: OutcomeFailed, Code: CodeQuota, HTTPStatus: 429}
	for _, field := range []string{"event", "outcome", "code", "status"} {
		d := valid
		switch field {
		case "event":
			d.Event = "secret-token"
		case "outcome":
			d.Outcome = "C:/private/replay.rofl"
		case "code":
			d.Code = "server error with personal data"
		case "status":
			d.HTTPStatus = 123456789
		}
		if err := s.RecordDiagnostic(d); err == nil {
			t.Fatalf("accepted arbitrary %s", field)
		}
	}
	if _, err := os.Stat(filepath.Join(s.dir, diagnosticName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid data created log: %v", err)
	}
	if err := s.RecordDiagnostic(valid); err != nil {
		t.Fatal(err)
	}
	rows := diagnosticRows(t, filepath.Join(s.dir, diagnosticName))
	if len(rows) != 1 || len(rows[0]) != 5 || rows[0]["event"] != "upload" || rows[0]["code"] != "quota" || rows[0]["httpStatus"] != float64(429) {
		t.Fatalf("unexpected record: %+v", rows)
	}
	when, err := time.Parse(time.RFC3339Nano, rows[0]["time"].(string))
	if err != nil || when.Location() != time.UTC {
		t.Fatalf("not a UTC timestamp: %v %v", when, err)
	}
}

func TestDiagnosticsBoundedRotation(t *testing.T) {
	s, err := OpenStore(t.TempDir(), "test-origin")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for range 7000 {
		if err := s.RecordDiagnostic(Diagnostic{Event: EventUpload, Outcome: OutcomeSucceeded}); err != nil {
			t.Fatal(err)
		}
	}
	retained := 0
	for _, name := range []string{diagnosticName, previousDiagnosticName} {
		path := filepath.Join(s.dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > diagnosticMaxBytes {
			t.Fatalf("%s exceeds limit: %d", name, info.Size())
		}
		retained += len(diagnosticRows(t, path))
	}
	if retained == 0 || retained >= 7000 {
		t.Fatalf("rotation did not discard old records: %d", retained)
	}
	files, err := os.ReadDir(s.dir)
	if err != nil || len(files) != 3 {
		t.Fatalf("expected two logs and lock: %v %v", files, err)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	files, err = os.ReadDir(s.dir)
	if err != nil || len(files) != 1 || files[0].Name() != "instance.lock" {
		t.Fatalf("clear retained rotated logs or lost lock: %v %v", files, err)
	}
}

func TestDiagnosticsConcurrentWritesClearAndClose(t *testing.T) {
	base := t.TempDir()
	s, err := OpenStore(base, "origin-a")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	other, err := OpenStore(base, "origin-b")
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	d := Diagnostic{Event: EventOperation, Outcome: OutcomeStarted}
	if err := other.RecordDiagnostic(d); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 100 {
				if err := s.RecordDiagnostic(d); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	wg.Wait()
	if rows := diagnosticRows(t, filepath.Join(s.dir, diagnosticName)); len(rows) != 1000 {
		t.Fatalf("lost/interleaved records: %d", len(rows))
	}
	for range 5 {
		wg.Go(func() {
			for range 20 {
				if err := s.RecordDiagnostic(d); err != nil {
					t.Error(err)
					return
				}
			}
		})
		wg.Go(func() {
			if err := s.Clear(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{diagnosticName, previousDiagnosticName} {
		if _, err := os.Stat(filepath.Join(s.dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("clear retained %s: %v", name, err)
		}
	}
	if got := diagnosticRows(t, filepath.Join(other.dir, diagnosticName)); len(got) != 1 {
		t.Fatal("clear touched another origin")
	}
	if second, err := OpenStore(base, "origin-a"); err == nil {
		second.Close()
		t.Fatal("clear lost instance lock")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordDiagnostic(d); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("wrote after releasing instance lock: %v", err)
	}
	if err := s.Clear(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("cleared after releasing instance lock: %v", err)
	}
}

func diagnosticRows(t *testing.T, path string) []map[string]any {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var rows []map[string]any
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var row map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatalf("broken log line: %v", err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}
