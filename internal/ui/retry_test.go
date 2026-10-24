package ui

import (
	"context"
	"errors"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/discovery"
	"strings"
	"testing"
	"time"
)

type retryFake struct {
	device string
	calls  []*desktop.Entry
	upload func(context.Context, *desktop.Entry) error
	lookup func(context.Context) (desktop.Session, error)
}

func (f *retryFake) DeviceSessionID() string { return f.device }
func (f *retryFake) Upload(ctx context.Context, e *desktop.Entry, _ desktop.Progress) error {
	f.calls = append(f.calls, e)
	return f.upload(ctx, e)
}
func (f *retryFake) Session(ctx context.Context) (desktop.Session, error) { return f.lookup(ctx) }
func retryFixture() (*retryFake, *retryRequest) {
	file := discovery.File{Path: "replay.rofl", Name: "replay.rofl", Size: 32}
	e := desktop.NewEntry(file)
	e.ReservationAttempted = true
	e.DeviceSessionID = connectionID
	e.Binding.UploadID = connectionID
	request := &retryRequest{entry: e, file: file, deviceID: connectionID}
	f := &retryFake{device: connectionID}
	f.upload = func(context.Context, *desktop.Entry) error {
		return &desktop.APIError{Status: 409, Code: "reservation_expired"}
	}
	f.lookup = func(context.Context) (desktop.Session, error) {
		s := desktop.Session{DeviceSessionID: connectionID}
		s.Quota.Account.Remaining = 1
		s.Quota.Global.Remaining = 1
		return s, nil
	}
	return f, request
}
func TestRetryOffersConfirmationWithoutReplacingReservation(t *testing.T) {
	f, request := retryFixture()
	key := request.entry.Key
	result := retrySelected(context.Background(), f, request, false, nil)
	if result.retryOffer != request || result.replacement != nil || len(f.calls) != 1 || request.entry.Key != key {
		t.Fatalf("reservation changed before confirmation: %+v", result)
	}
	// Declining requires no further operation. A canceled worker must suppress a queued offer.
	ctx, cancel := context.WithCancel(context.Background())
	canceled := runWork(ctx, func(context.Context) update { cancel(); return result })
	if canceled.retryOffer != nil || canceled.replacement != nil {
		t.Fatal("canceled worker retained confirmation")
	}
}
func TestConfirmedRetryRechecksBeforeCreatingNewKey(t *testing.T) {
	f, request := retryFixture()
	f.upload = func(_ context.Context, e *desktop.Entry) error {
		if e == request.entry {
			return &desktop.APIError{Status: 409, Code: "reservation_expired"}
		}
		if !e.NeedsReservation() || e.Key == request.entry.Key || e.File != request.file {
			t.Fatal("new attempt lost fresh identity or selection")
		}
		e.ReservationAttempted = true
		e.DeviceSessionID = connectionID
		e.Result.State = "completed"
		return nil
	}
	result := retrySelected(context.Background(), f, request, true, nil)
	if len(f.calls) != 2 || f.calls[0] != request.entry || result.replacement != f.calls[1] || result.message != "완료" {
		t.Fatalf("new attempt not completed: %+v", result)
	}
	if request.entry.Binding.UploadID != connectionID {
		t.Fatal("old binding mutated")
	}
}
func TestConfirmedRetryDoesNotReplaceRecoveredOrUncertainResults(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"recovered", nil}, {"network", desktop.ErrNetwork}, {"auth", desktop.ErrLogin},
		{"processing", errors.New("서버 처리 중")}, {"storage", &desktop.APIError{Status: 503, Code: "storage_unavailable"}},
		{"forbidden", &desktop.APIError{Status: 403, Code: "forbidden"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, request := retryFixture()
			// A stale failed result must never authorize replacing a reservation after a lookup failure.
			request.entry.Result.State = "failed"
			f.upload = func(context.Context, *desktop.Entry) error { return tc.err }
			f.lookup = func(context.Context) (desktop.Session, error) {
				t.Fatal("unexpected quota lookup")
				return desktop.Session{}, nil
			}
			result := retrySelected(context.Background(), f, request, true, nil)
			if len(f.calls) != 1 || result.replacement != nil || result.retryOffer != nil {
				t.Fatal("replaced recovered or uncertain entry")
			}
		})
	}
}
func TestConfirmedRetryStopsAtQuotaDeviceOrCancellation(t *testing.T) {
	for _, mode := range []string{"quota", "account quota", "device", "session device", "session error", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			f, request := retryFixture()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			original := f.lookup
			f.lookup = func(ctx context.Context) (desktop.Session, error) {
				s, _ := original(ctx)
				switch mode {
				case "quota":
					s.Quota.Global.Remaining = 0
				case "account quota":
					s.Quota.Account.Remaining = 0
				case "device":
					f.device = "other"
				case "session device":
					s.DeviceSessionID = "other"
				case "session error":
					return s, desktop.ErrNetwork
				case "cancel":
					cancel()
				}
				return s, nil
			}
			result := retrySelected(ctx, f, request, true, nil)
			if len(f.calls) != 1 || result.replacement != nil {
				t.Fatalf("replaced after %s", mode)
			}
		})
	}
	f, request := retryFixture()
	f.device = "different"
	result := retrySelected(context.Background(), f, request, true, nil)
	if len(f.calls) != 0 || result.replacement != nil {
		t.Fatal("used old confirmation on different device")
	}
}
func TestCanceledNewAttemptRetainsAttemptedIdentity(t *testing.T) {
	f, request := retryFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.upload = func(_ context.Context, e *desktop.Entry) error {
		if e == request.entry {
			return &desktop.APIError{Status: 409, Code: "reservation_expired"}
		}
		e.ReservationAttempted = true
		e.DeviceSessionID = connectionID
		cancel()
		return context.Canceled
	}
	result := runWork(ctx, func(ctx context.Context) update { return retrySelected(ctx, f, request, true, nil) })
	if result.replacement == nil || !result.replacement.ReservationAttempted || result.replacement.Key == request.entry.Key || !strings.Contains(result.message, "취소됨") {
		t.Fatalf("lost uncertain new identity: %+v", result)
	}
}

func TestCancellationAfterWorkerCompletionSuppressesQueuedOffer(t *testing.T) {
	f, request := retryFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := runWork(ctx, func(ctx context.Context) update {
		return retrySelected(ctx, f, request, false, nil)
	})
	if ready.retryOffer == nil {
		t.Fatal("worker did not produce confirmation offer")
	}
	// The UI receives cancellation after the final result has been queued.
	updates := make(chan update, 1)
	updates <- ready
	cancel()
	delivered := suppressCanceledNavigation(<-updates)
	if delivered.retryOffer != nil || delivered.picker || delivered.replacement != nil {
		t.Fatal("late cancellation still requests navigation")
	}
	// A completed new attempt must keep its binding even when delivery is late.
	next := desktop.NewEntry(request.file)
	next.ReservationAttempted = true
	delivered = suppressCanceledNavigation(update{replacement: next, clearEntries: true, signedOut: true})
	if delivered.replacement != next || !delivered.clearEntries || !delivered.signedOut {
		t.Fatal("late cancellation erased committed state")
	}
}

func TestRetryOfUnreservedFileStillChecksQuota(t *testing.T) {
	for _, remaining := range []int{0, 1} {
		t.Run(string(rune('0'+remaining)), func(t *testing.T) {
			f, request := retryFixture()
			request.entry = desktop.NewEntry(request.file)
			original := f.lookup
			lookups := 0
			f.lookup = func(ctx context.Context) (desktop.Session, error) {
				lookups++
				s, err := original(ctx)
				s.Quota.Account.Remaining = remaining
				return s, err
			}
			f.upload = func(context.Context, *desktop.Entry) error { return nil }
			result := retrySelected(context.Background(), f, request, false, nil)
			if lookups != 1 || len(f.calls) != remaining || result.retryOffer != nil || result.replacement != nil {
				t.Fatalf("fresh retry skipped quota: lookups=%d calls=%d result=%+v", lookups, len(f.calls), result)
			}
		})
	}
}

func TestRetrySelectionRefreshesOnlyUnattemptedFile(t *testing.T) {
	for _, attempted := range []bool{false, true} {
		f, previous := retryFixture()
		old := desktop.NewEntry(previous.file)
		old.ReservationAttempted = attempted
		if attempted {
			old.DeviceSessionID = connectionID
			old.Binding.UploadID = connectionID
		}
		entries := map[string]*desktop.Entry{old.File.Path: old}
		current := old.File
		current.Size++
		request := selectRetry(entries, current, connectionID)
		if request == nil || request.file != current || request.entry != entries[current.Path] {
			t.Fatal("missing refreshed row")
		}
		if attempted {
			if request.entry != old || request.entry.File != old.File {
				t.Fatal("changed attempted reservation identity")
			}
		} else {
			f.upload = func(_ context.Context, e *desktop.Entry) error {
				if e.File != current || e.Key == old.Key {
					t.Fatal("retry uses stale local metadata")
				}
				return nil
			}
			result := retrySelected(context.Background(), f, request, false, nil)
			if len(f.calls) != 1 || result.message != "완료" {
				t.Fatal("fresh selection could not retry")
			}
		}
	}
	entries := map[string]*desktop.Entry{}
	file := discovery.File{Path: "replay.rofl"}
	if selectRetry(entries, file, connectionID) != nil {
		t.Fatal("retry created an unselected entry")
	}
	for _, state := range []string{"completed", "duplicate"} {
		entry := desktop.NewEntry(file)
		entry.Result.State = state
		entries[file.Path] = entry
		if selectRetry(entries, file, connectionID) != nil {
			t.Fatal("retry offered successful result")
		}
	}
}

func TestRateLimitedRetryRefreshesQuotaWithoutStartingNewReservation(t *testing.T) {
	for _, mode := range []string{"success", "lookup failure", "different device"} {
		t.Run(mode, func(t *testing.T) {
			f, request := retryFixture()
			f.upload = func(context.Context, *desktop.Entry) error {
				return &desktop.APIError{Status: 429, Code: "quota_exhausted", RetryAfter: 30 * time.Second}
			}
			lookups := 0
			f.lookup = func(context.Context) (desktop.Session, error) {
				lookups++
				s := desktop.Session{DeviceSessionID: connectionID}
				if mode == "lookup failure" {
					return s, desktop.ErrNetwork
				}
				if mode == "different device" {
					s.DeviceSessionID = "other"
				}
				return s, nil
			}
			result := retrySelected(context.Background(), f, request, true, nil)
			if lookups != 1 || len(f.calls) != 1 || result.retryOffer != nil || result.replacement != nil {
				t.Fatalf("429 started a new attempt: %+v", result)
			}
			if (result.session != nil) != (mode == "success") {
				t.Fatalf("wrong quota snapshot after %s: %+v", mode, result)
			}
			if !strings.Contains(result.message, "최소 30초") {
				t.Fatalf("429 delay was lost: %s", result.message)
			}
		})
	}
}

func TestRefreshThrottleDoesNotReplayThroughQuotaLookup(t *testing.T) {
	for _, afterReplacement := range []bool{false, true} {
		t.Run(map[bool]string{false: "existing", true: "replacement"}[afterReplacement], func(t *testing.T) {
			f, request := retryFixture()
			key := request.entry.Key
			lookups := 0
			lookup := f.lookup
			f.lookup = func(ctx context.Context) (desktop.Session, error) { lookups++; return lookup(ctx) }
			f.upload = func(_ context.Context, e *desktop.Entry) error {
				if afterReplacement && e == request.entry {
					return &desktop.APIError{Status: 409, Code: "reservation_expired"}
				}
				return &desktop.APIError{Status: 429, Code: "rate_limited", RetryAfter: time.Minute}
			}
			result := retrySelected(context.Background(), f, request, afterReplacement, nil)
			wantLookups := 0
			if afterReplacement {
				wantLookups = 1
			}
			if lookups != wantLookups || result.retryOffer != nil || request.entry.Key != key || !strings.Contains(result.message, "60초") {
				t.Fatalf("throttling replayed through a quota lookup or lost the original entry: %+v", result)
			}
		})
	}
}
