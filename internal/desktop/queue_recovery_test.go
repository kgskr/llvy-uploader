package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"llvy-uploader/internal/replay"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func recoveryBinding() Reservation {
	return Reservation{UploadID: fixtureID, Nonce: strings.Repeat("A", 43), ExpiresAt: time.Now().Add(30 * time.Minute)}
}

func completedResult() Result {
	return Result{UploadID: fixtureID, State: "completed", GameID: fixtureID, GamePath: "/games/" + fixtureID}
}

func pendingEntry(t *testing.T) *Entry {
	t.Helper()
	e := NewEntry(fileFixture(t))
	h, err := replay.Open(e.File)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	hash, err := h.Hash(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	e.Metadata = Metadata{Filename: e.File.Name, Size: e.File.Size, LastModified: e.File.ModTime.UnixMilli(), SHA256: hash}
	e.Binding = recoveryBinding()
	e.ReservationAttempted = true
	e.DeviceSessionID = fixtureID
	return e
}

func errorResponse(status int, code string) *http.Response {
	r := response(map[string]any{"error": map[string]string{"code": code}})
	r.StatusCode = status
	return r
}

func changeOriginal(t *testing.T, e *Entry, change string) {
	t.Helper()
	var err error
	switch change {
	case "moved":
		err = os.Rename(e.File.Path, e.File.Path+".moved")
	case "deleted":
		err = os.Remove(e.File.Path)
	case "changed":
		// Preserve size and mtime so reopening would fail the saved hash check.
		err = os.WriteFile(e.File.Path, []byte(strings.Repeat("X", int(e.File.Size))), 0600)
		if err == nil {
			err = os.Chtimes(e.File.Path, e.File.ModTime, e.File.ModTime)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestPendingPUTRecoveryDoesNotRequireOriginal(t *testing.T) {
	for _, change := range []string{"moved", "deleted", "changed"} {
		t.Run(change, func(t *testing.T) {
			c, _ := New(testOrigin, false, nil)
			c.credentials = credentials()
			e := NewEntry(fileFixture(t))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			reserves, puts, processes, statuses := 0, 0, 0, 0
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				switch {
				case r.Method == http.MethodPut:
					puts++
					_, _ = io.ReadAll(r.Body)
					return response(nil), nil
				case strings.HasSuffix(r.URL.Path, "/uploads"):
					reserves++
					return response(recoveryBinding()), nil
				case strings.HasSuffix(r.URL.Path, "/transfer"):
					return response(descriptor()), nil
				case strings.HasSuffix(r.URL.Path, "/process"):
					processes++
					if processes == 1 {
						cancel()
						return nil, context.Canceled
					}
					return response(completedResult()), nil
				default:
					statuses++
					return response(Result{UploadID: fixtureID, State: "pending"}), nil
				}
			})
			if err := c.Upload(ctx, e, nil); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			changeOriginal(t, e, change)
			statuses = 0
			if err := c.Upload(context.Background(), e, func(_ *Entry, stage string, _, _ int64) {
				if stage == "파일 확인" {
					t.Error("accessed original before server recovery")
				}
			}); err != nil {
				t.Fatal(err)
			}
			if reserves != 1 || puts != 1 || processes != 2 || statuses != 1 || e.Result.State != "completed" {
				t.Fatalf("reserves=%d puts=%d processes=%d statuses=%d result=%+v", reserves, puts, processes, statuses, e.Result)
			}
		})
	}
}

func TestPendingStorageUncertaintyDoesNotOpenOrPut(t *testing.T) {
	c, _ := New(testOrigin, false, nil)
	c.credentials = credentials()
	e := pendingEntry(t)
	changeOriginal(t, e, "deleted")
	processes, statuses := 0, 0
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/process") {
			processes++
			return errorResponse(503, "storage_unavailable"), nil
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected mutation %s %s", r.Method, r.URL.Path)
		}
		statuses++
		return response(Result{UploadID: fixtureID, State: "pending"}), nil
	})
	err := c.Upload(context.Background(), e, func(_ *Entry, stage string, _, _ int64) {
		if stage == "파일 확인" {
			t.Error("opened original under uncertain storage state")
		}
	})
	if err == nil || NewAttemptAllowed(err) || processes != 3 || statuses != 4 {
		t.Fatalf("err=%v processes=%d statuses=%d", err, processes, statuses)
	}
}

func TestConfirmedAbsenceValidatesOriginalBeforeRetransmission(t *testing.T) {
	for _, change := range []string{"unchanged", "deleted", "changed"} {
		t.Run(change, func(t *testing.T) {
			c, _ := New(testOrigin, false, nil)
			c.credentials = credentials()
			e := pendingEntry(t)
			metadata, key, binding := e.Metadata, e.Key, e.Binding
			changeOriginal(t, e, change)
			processes, puts, transfers := 0, 0, 0
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				switch {
				case r.Method == http.MethodPut:
					puts++
					return response(nil), nil
				case strings.HasSuffix(r.URL.Path, "/process"):
					processes++
					if processes == 1 {
						return errorResponse(409, "object_not_uploaded"), nil
					}
					return response(completedResult()), nil
				case strings.HasSuffix(r.URL.Path, "/transfer"):
					transfers++
					return response(descriptor()), nil
				default:
					if r.Method != http.MethodGet {
						t.Fatal("created replacement reservation")
					}
					return response(Result{UploadID: fixtureID, State: "pending"}), nil
				}
			})
			err := c.Upload(context.Background(), e, nil)
			if change == "unchanged" {
				if err != nil || puts != 1 || transfers != 1 || processes != 2 {
					t.Fatalf("err=%v puts=%d transfers=%d processes=%d", err, puts, transfers, processes)
				}
			} else if err == nil || NewAttemptAllowed(err) || puts != 0 || transfers != 0 || processes != 1 {
				t.Fatalf("err=%v puts=%d transfers=%d processes=%d", err, puts, transfers, processes)
			}
			if change == "changed" && !errors.Is(err, replay.ErrChanged) {
				t.Fatalf("changed hash was not rejected: %v", err)
			}
			if e.Metadata != metadata || e.Key != key || e.Binding != binding {
				t.Fatal("changed attempted reservation identity")
			}
		})
	}
}

func TestLostReservationResponseRecoversBeforeOriginalAccess(t *testing.T) {
	c, _ := New(testOrigin, false, nil)
	c.credentials = credentials()
	e := NewEntry(fileFixture(t))
	lost := true
	reserves, processes := 0, 0
	var metadata Metadata
	key := e.Key
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/uploads"):
			reserves++
			var request Metadata
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if reserves == 1 {
				metadata = request
			}
			if request != metadata || r.Header.Get("Idempotency-Key") != key {
				t.Fatal("lost response recovery changed reservation identity")
			}
			if lost {
				return nil, io.ErrUnexpectedEOF
			}
			return response(recoveryBinding()), nil
		case strings.HasSuffix(r.URL.Path, "/process"):
			processes++
			return response(completedResult()), nil
		default:
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			return response(Result{UploadID: fixtureID, State: "pending"}), nil
		}
	})
	if err := c.Upload(context.Background(), e, nil); !errors.Is(err, ErrNetwork) {
		t.Fatal(err)
	}
	changeOriginal(t, e, "deleted")
	lost = false
	ready, deferred := QuotaSelection([]*Entry{e}, 0)
	if len(ready) != 1 || deferred != 0 {
		t.Fatal("zero quota blocked existing reservation")
	}
	if err := c.Upload(context.Background(), ready[0], func(_ *Entry, stage string, _, _ int64) {
		if stage == "파일 확인" {
			t.Error("lost reservation recovery opened original")
		}
	}); err != nil {
		t.Fatal(err)
	}
	if reserves != 4 || processes != 1 || e.Metadata != metadata || e.Key != key {
		t.Fatalf("reserves=%d processes=%d entry=%+v", reserves, processes, e)
	}
}

func TestNewAttemptAllowedRequiresConfirmedTerminalOutcome(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"failed result", resultError(Result{State: "failed", ErrorCode: "download_failed"}), true},
		{"unknown result", resultError(Result{State: "outcome_unknown"}), true},
		{"wrapped result", fmt.Errorf("recovery: %w", resultError(Result{State: "failed"})), true},
		{"expired reservation", &APIError{Status: 409, Code: "reservation_expired"}, true},
		{"expiry with wrong status", &APIError{Status: 503, Code: "reservation_expired"}, false},
		{"uncertain API response", &APIError{Status: 503, Code: "outcome_unknown"}, false},
		{"pending", resultError(Result{State: "pending"}), false},
		{"processing", resultError(Result{State: "processing"}), false},
		{"completed", resultError(completedResult()), false},
		{"network", ErrNetwork, false},
		{"auth", ErrLogin, false},
		{"canceled", context.Canceled, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewAttemptAllowed(tc.err); got != tc.want {
				t.Fatalf("NewAttemptAllowed(%v)=%v", tc.err, got)
			}
		})
	}
}

func TestStaleTerminalResultDoesNotAuthorizeReplacementOnStatusFailure(t *testing.T) {
	c, _ := New(testOrigin, false, nil)
	c.credentials = credentials()
	e := pendingEntry(t)
	e.Result = Result{UploadID: fixtureID, State: "failed", ErrorCode: "download_failed"}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return errorResponse(403, "forbidden"), nil
	})
	err := c.Upload(context.Background(), e, nil)
	if !PauseQueue(err) || NewAttemptAllowed(err) {
		t.Fatalf("stale result authorized replacement after status failure: %v", err)
	}
}

func TestRecoveryStatusAuthorityFailurePausesQueue(t *testing.T) {
	for _, status := range []int{403, 429} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			c, _ := New(testOrigin, false, nil)
			c.credentials = credentials()
			e := pendingEntry(t)
			statuses := 0
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/process") {
					return errorResponse(503, "storage_unavailable"), nil
				}
				statuses++
				if statuses == 1 {
					return response(Result{UploadID: fixtureID, State: "pending"}), nil
				}
				return errorResponse(status, map[int]string{403: "forbidden", 429: "quota_exhausted"}[status]), nil
			})
			err := c.Upload(context.Background(), e, nil)
			var api *APIError
			if !errors.As(err, &api) || api.Status != status || !PauseQueue(err) || NewAttemptAllowed(err) {
				t.Fatalf("masked status authority failure: %v", err)
			}
		})
	}
}
