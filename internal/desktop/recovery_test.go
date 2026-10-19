package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"llvy-uploader/internal/discovery"
	"llvy-uploader/internal/replay"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func response(value any) *http.Response {
	b, _ := json.Marshal(value)
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(b)))}
}

func TestProcessRefreshesBeforeClaimAndRejectsExpiringDevice(t *testing.T) {
	for _, expiring := range []bool{false, true} {
		t.Run(map[bool]string{false: "refresh", true: "absolute expiry"}[expiring], func(t *testing.T) {
			c, _ := New(ProductOrigin, false, nil)
			next := credentials()
			if expiring {
				next.DeviceExpiresAt = time.Now().Add(time.Minute)
				next.AccessExpiresAt = next.DeviceExpiresAt
			}
			c.credentials = next
			c.credentials.AccessExpiresAt = time.Now().Add(35 * time.Second)
			refresh, process := 0, 0
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/refresh") {
					refresh++
					return response(next), nil
				}
				process++
				if time.Until(c.credentials.AccessExpiresAt) < 2*time.Minute {
					t.Error("started claim with short credential")
				}
				return response(Result{UploadID: fixtureID, State: "pending"}), nil
			})
			_, err := c.Process(context.Background(), Reservation{UploadID: fixtureID})
			if refresh != 1 {
				t.Fatalf("refreshes=%d", refresh)
			}
			if expiring {
				if !errors.Is(err, ErrLogin) || process != 0 || c.credentials.RefreshToken != "" {
					t.Fatalf("err=%v requests=%d", err, process)
				}
			} else if err != nil || process != 1 {
				t.Fatalf("err=%v requests=%d", err, process)
			}
		})
	}
}

func TestLostReservationRecoversAtZeroQuotaWithSameIdentity(t *testing.T) {
	c, _ := New(ProductOrigin, false, nil)
	c.credentials = credentials()
	e := NewEntry(fileFixture(t))
	lost := true
	calls := 0
	key := ""
	var metadata Metadata
	binding := Reservation{UploadID: fixtureID, Nonce: strings.Repeat("A", 43), ExpiresAt: time.Now().Add(30 * time.Minute)}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/uploads") {
			calls++
			var m Metadata
			_ = json.NewDecoder(r.Body).Decode(&m)
			if key == "" {
				key = r.Header.Get("Idempotency-Key")
				metadata = m
			}
			if key != r.Header.Get("Idempotency-Key") || metadata != m {
				t.Error("reservation identity changed")
			}
			if lost {
				return nil, io.ErrUnexpectedEOF
			}
			return response(binding), nil
		}
		// Stop after proving that the same reservation was recovered, before any PUT.
		return nil, context.Canceled
	})
	if err := c.Upload(context.Background(), e, nil); !errors.Is(err, ErrNetwork) {
		t.Fatal(err)
	}
	if calls != 3 || !e.ReservationAttempted || e.Binding.UploadID != "" {
		t.Fatalf("unexpected lost response state: %+v", e)
	}
	ready, deferred := QuotaSelection([]*Entry{e}, 0)
	if len(ready) != 1 || deferred != 0 {
		t.Fatal("recovery blocked by quota")
	}
	lost = false
	_ = c.Upload(context.Background(), ready[0], nil)
	if calls != 4 || e.Binding.UploadID != binding.UploadID || e.Binding.Nonce != binding.Nonce || !e.Binding.ExpiresAt.Equal(binding.ExpiresAt) {
		t.Fatalf("reservation not recovered: calls=%d %+v", calls, e.Binding)
	}
}

func TestQuotaSelectionKeepsRecoveriesWhenFreshBatchIsTooLarge(t *testing.T) {
	recovery := NewEntry(discovery.File{Path: "recovery"})
	recovery.ReservationAttempted = true
	fresh := NewEntry(discovery.File{Path: "fresh"})
	ready, deferred := QuotaSelection([]*Entry{fresh, recovery}, 0)
	if len(ready) != 1 || ready[0] != recovery || deferred != 1 {
		t.Fatal("mixed selection lost recovery")
	}
	ready, deferred = QuotaSelection([]*Entry{fresh, recovery}, 1)
	if len(ready) != 2 || deferred != 0 {
		t.Fatal("valid new selection blocked")
	}
}

func TestReselectionUpdatesOnlyUnattemptedFiles(t *testing.T) {
	file := fileFixture(t)
	entries := map[string]*Entry{}
	old := SelectEntry(entries, file)
	if err := os.WriteFile(file.Path, []byte("changed replay fixture with a different size"), 0600); err != nil {
		t.Fatal(err)
	}
	c, _ := New(ProductOrigin, false, nil)
	c.credentials = credentials()
	if err := c.Upload(context.Background(), old, nil); !errors.Is(err, replay.ErrChanged) {
		t.Fatal(err)
	}
	current, _, err := (discovery.OSReader{}).Inspect(file.Path)
	if err != nil {
		t.Fatal(err)
	}
	next := SelectEntry(entries, current)
	if next == old || next.Key == old.Key {
		t.Fatal("stale selection retained")
	}
	handle, err := replay.Open(next.File)
	if err != nil {
		t.Fatal(err)
	}
	handle.Close()
	next.ReservationAttempted = true
	changed := current
	changed.Size++
	if selected := SelectEntry(entries, changed); selected != next || selected.File.Size != current.Size {
		t.Fatal("uncertain reservation identity changed")
	}
}

func TestCanceledLogoutRemovesCredentials(t *testing.T) {
	c, _ := New(ProductOrigin, false, nil)
	c.credentials = credentials()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { cancel(); return nil, r.Context().Err() })
	if err := c.Logout(ctx); err == nil {
		t.Fatal("missing remote revoke warning")
	}
	if c.credentials.RefreshToken != "" {
		t.Fatal("local credentials remain")
	}
}
