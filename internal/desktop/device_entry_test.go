package desktop

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestEntriesCannotCrossDeviceSessions(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(map[bool]string{false: "lost reservation response", true: "known reservation"}[known], func(t *testing.T) {
			c, _ := New(ProductOrigin, false, nil)
			c.credentials = credentials()
			entry := NewEntry(fileFixture(t))
			entry.ReservationAttempted = true
			entry.DeviceSessionID = c.DeviceSessionID()
			if known {
				entry.Binding = Reservation{UploadID: fixtureID, Nonce: strings.Repeat("A", 43), ExpiresAt: time.Now().Add(time.Minute)}
			}
			next := credentials()
			next.DeviceSessionID = "22222222-2222-4222-8222-222222222222"
			if err := c.accept(next, false); err != nil {
				t.Fatal(err)
			}
			calls := 0
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return response(Reservation{UploadID: fixtureID, Nonce: strings.Repeat("A", 43), ExpiresAt: time.Now().Add(time.Minute)}), nil
			})
			if err := c.Upload(context.Background(), entry, nil); !errors.Is(err, ErrDeviceChanged) || !PauseQueue(err) {
				t.Fatalf("unexpected recovery: %v", err)
			}
			if calls != 0 {
				t.Fatal("old device entry sent a request")
			}
			// Explicit fresh selection gets a fresh key and owns the new device's reservation.
			fresh := NewEntry(entry.File)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/uploads") {
					calls++
					if r.Header.Get("Idempotency-Key") == entry.Key {
						t.Error("reused old device's key")
					}
					return response(Reservation{UploadID: fixtureID, Nonce: strings.Repeat("A", 43), ExpiresAt: time.Now().Add(time.Minute)}), nil
				}
				cancel()
				return nil, ctx.Err()
			})
			_ = c.Upload(ctx, fresh, nil)
			if calls != 1 || fresh.DeviceSessionID != next.DeviceSessionID {
				t.Fatal("fresh reservation not bound to new device")
			}
		})
	}
}

func TestSameDeviceRefreshPreservesEntry(t *testing.T) {
	c, _ := New(ProductOrigin, false, nil)
	next := credentials()
	c.credentials = next
	c.credentials.AccessExpiresAt = time.Now().Add(-time.Minute)
	entry := NewEntry(fileFixture(t))
	entry.ReservationAttempted = true
	entry.DeviceSessionID = fixtureID
	entry.Binding.UploadID = fixtureID
	refresh, status := 0, 0
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/refresh") {
			refresh++
			return response(next), nil
		}
		status++
		return response(Result{UploadID: fixtureID, State: "completed", GameID: fixtureID, GamePath: "/games/" + fixtureID}), nil
	})
	if err := c.Upload(context.Background(), entry, nil); err != nil {
		t.Fatal(err)
	}
	if refresh != 1 || status != 1 || entry.DeviceSessionID != fixtureID {
		t.Fatal("same-device recovery changed identity")
	}
}
