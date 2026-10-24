package platform

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const (
	diagnosticName         = "diagnostics.jsonl"
	previousDiagnosticName = "diagnostics.1.jsonl"
	diagnosticMaxBytes     = 256 << 10
)

type DiagnosticEvent string
type DiagnosticOutcome string
type DiagnosticCode string

const (
	EventStartup        DiagnosticEvent   = "startup"
	EventShutdown       DiagnosticEvent   = "shutdown"
	EventOperation      DiagnosticEvent   = "operation"
	EventUpload         DiagnosticEvent   = "upload"
	EventAuthentication DiagnosticEvent   = "authentication"
	OutcomeStarted      DiagnosticOutcome = "started"
	OutcomeSucceeded    DiagnosticOutcome = "succeeded"
	OutcomeFailed       DiagnosticOutcome = "failed"
	OutcomeCanceled     DiagnosticOutcome = "canceled"

	CodeNone           DiagnosticCode = ""
	CodeAPI            DiagnosticCode = "api"
	CodeAuthentication DiagnosticCode = "authentication"
	CodeQuota          DiagnosticCode = "quota"
	CodeNetwork        DiagnosticCode = "network"
	CodeLocal          DiagnosticCode = "local"
	CodeUnknown        DiagnosticCode = "unknown"
)

// Diagnostic intentionally has no free-text, identifier or path field.
// Typed strings are still checked against fixed allowlists before writing.
type Diagnostic struct {
	Event      DiagnosticEvent   `json:"event"`
	Outcome    DiagnosticOutcome `json:"outcome"`
	Code       DiagnosticCode    `json:"code,omitempty"`
	HTTPStatus int               `json:"httpStatus,omitempty"`
}

func (d Diagnostic) valid() bool {
	switch d.Event {
	case EventStartup, EventShutdown, EventOperation, EventUpload, EventAuthentication:
	default:
		return false
	}
	switch d.Outcome {
	case OutcomeStarted, OutcomeSucceeded, OutcomeFailed, OutcomeCanceled:
	default:
		return false
	}
	switch d.Code {
	case CodeNone, CodeAPI, CodeAuthentication, CodeQuota, CodeNetwork, CodeLocal, CodeUnknown:
	default:
		return false
	}
	return d.HTTPStatus == 0 || d.HTTPStatus >= 100 && d.HTTPStatus <= 599
}

// RecordDiagnostic writes a bounded, redacted local record. A logging failure
// must not replace the result of the operation being diagnosed.
func (s *Store) RecordDiagnostic(d Diagnostic) error {
	if !d.valid() {
		return errors.New("invalid diagnostic category")
	}
	record := struct {
		Time string `json:"time"`
		Diagnostic
	}{Time: time.Now().UTC().Format(time.RFC3339Nano), Diagnostic: d}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return os.ErrClosed
	}
	path := filepath.Join(s.dir, diagnosticName)
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil && info.Size()+int64(len(data)) > diagnosticMaxBytes {
		// replaceFile also supports replacing an existing generation on Windows.
		if info.Size() > diagnosticMaxBytes {
			// Do not carry externally enlarged files into the bounded backup.
			if err := os.Remove(path); err != nil {
				return err
			}
		} else if err := replaceFile(path, filepath.Join(s.dir, previousDiagnosticName)); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
