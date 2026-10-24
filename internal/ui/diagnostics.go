package ui

import (
	"context"
	"errors"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/platform"
	"net"
	"os"
)

type diagnosticSink interface {
	RecordDiagnostic(platform.Diagnostic) error
}

// Never pass error strings, identifiers, paths or HTTP response bodies to storage.
func diagnosticResult(event platform.DiagnosticEvent, err error) platform.Diagnostic {
	d := platform.Diagnostic{Event: event, Outcome: platform.OutcomeSucceeded}
	if err == nil {
		return d
	}
	d.Outcome = platform.OutcomeFailed
	d.Code = platform.CodeUnknown
	var api *desktop.APIError
	var network net.Error
	var local *os.PathError
	switch {
	case errors.Is(err, context.Canceled):
		d.Outcome = platform.OutcomeCanceled
		d.Code = platform.CodeNone
	case errors.Is(err, desktop.ErrLogin):
		d.Code = platform.CodeAuthentication
	case errors.Is(err, desktop.ErrNetwork):
		d.Code = platform.CodeNetwork
	case errors.As(err, &api):
		d.Code = platform.CodeAPI
		d.HTTPStatus = api.Status
		if api.Status == 429 {
			d.Code = platform.CodeQuota
		} else if api.Status == 401 || api.Status == 403 {
			d.Code = platform.CodeAuthentication
		}
	case errors.As(err, &network):
		d.Code = platform.CodeNetwork
	case errors.As(err, &local):
		d.Code = platform.CodeLocal
	}
	return d
}
func recordResult(sink diagnosticSink, event platform.DiagnosticEvent, err error) {
	// Diagnostics are best effort: a full disk must not change the upload result.
	_ = sink.RecordDiagnostic(diagnosticResult(event, err))
}

type diagnosticClient struct {
	*desktop.Client
	sink diagnosticSink
}

func (c diagnosticClient) Upload(ctx context.Context, e *desktop.Entry, progress desktop.Progress) error {
	_ = c.sink.RecordDiagnostic(platform.Diagnostic{Event: platform.EventUpload, Outcome: platform.OutcomeStarted})
	err := c.Client.Upload(ctx, e, progress)
	recordResult(c.sink, platform.EventUpload, err)
	return err
}
