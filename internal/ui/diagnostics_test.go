package ui

import (
	"context"
	"encoding/json"
	"errors"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/platform"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestDiagnosticErrorsDoNotRetainSecrets(t *testing.T) {
	secret := "refresh-secret/private/replay.rofl"
	cases := []struct {
		err     error
		code    platform.DiagnosticCode
		outcome platform.DiagnosticOutcome
		status  int
	}{
		{nil, platform.CodeNone, platform.OutcomeSucceeded, 0},
		{context.Canceled, platform.CodeNone, platform.OutcomeCanceled, 0},
		{desktop.ErrLogin, platform.CodeAuthentication, platform.OutcomeFailed, 0},
		{desktop.ErrNetwork, platform.CodeNetwork, platform.OutcomeFailed, 0},
		{&desktop.APIError{Status: 429, Code: secret}, platform.CodeQuota, platform.OutcomeFailed, 429},
		{&desktop.APIError{Status: 403, Code: secret}, platform.CodeAuthentication, platform.OutcomeFailed, 403},
		{&os.PathError{Op: "open", Path: secret, Err: os.ErrNotExist}, platform.CodeLocal, platform.OutcomeFailed, 0},
		{&url.Error{Op: "GET", URL: secret, Err: errors.New(secret)}, platform.CodeNetwork, platform.OutcomeFailed, 0},
		{errors.New(secret), platform.CodeUnknown, platform.OutcomeFailed, 0},
	}
	for _, tt := range cases {
		d := diagnosticResult(platform.EventUpload, tt.err)
		if d.Code != tt.code || d.Outcome != tt.outcome || d.HTTPStatus != tt.status {
			t.Fatalf("wrong category: %+v", d)
		}
		raw, _ := json.Marshal(d)
		if strings.Contains(string(raw), secret) {
			t.Fatal("sensitive diagnostic")
		}
	}
}
