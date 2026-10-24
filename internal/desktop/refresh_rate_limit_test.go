package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type refreshTestVault struct {
	value          string
	saves, deletes int
}

func (v *refreshTestVault) LoadRefresh() (string, error)   { return v.value, nil }
func (v *refreshTestVault) SaveRefresh(value string) error { v.value = value; v.saves++; return nil }
func (v *refreshTestVault) DeleteRefresh() error           { v.value = ""; v.deletes++; return nil }

func TestRefreshRateLimitPreservesCredentialsUntilExplicitRetry(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		t.Run(map[bool]string{false: "memory", true: "persistent"}[persistent], func(t *testing.T) {
			old := credentials()
			old.AccessExpiresAt = time.Now().Add(-time.Minute)
			next := credentials()
			next.DeviceExpiresAt = old.DeviceExpiresAt
			next.RefreshToken = "refresh." + strings.Repeat("C", 43)
			vault := &refreshTestVault{}
			if persistent {
				vault.value = old.RefreshToken
			}
			refreshes, lookups := 0, 0
			c := connected(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "auth/refresh") {
					refreshes++
					var body map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["refreshToken"] != old.RefreshToken {
						t.Error("refresh did not retain the unconsumed credential")
					}
					if refreshes == 1 {
						w.Header().Set("Retry-After", "60")
						w.WriteHeader(429)
						jsonResponse(w, map[string]any{"error": map[string]string{"code": "rate_limited"}})
						return
					}
					jsonResponse(w, next)
					return
				}
				lookups++
				jsonResponse(w, map[string]bool{"ok": true})
			})
			c.credentials, c.persistent, c.vault = old, persistent, vault
			err := c.authorized(context.Background(), "GET", "session", nil, nil, "")
			var api *APIError
			if !errors.As(err, &api) || api.Code != "rate_limited" || api.Status != 429 || api.RetryAfter != time.Minute {
				t.Fatalf("lost definitive throttling response: %v", err)
			}
			if c.credentials != old || c.persistent != persistent || !c.HasCredentials() || c.DeviceSessionID() != old.DeviceSessionID || vault.saves != 0 || vault.deletes != 0 || refreshes != 1 || lookups != 0 {
				t.Fatal("throttling changed credentials, device identity or retried automatically")
			}
			if persistent && vault.value != old.RefreshToken {
				t.Fatal("lost saved refresh token")
			}
			if err := c.authorized(context.Background(), "GET", "session", nil, nil, ""); err != nil {
				t.Fatal(err)
			}
			if c.credentials != next || refreshes != 2 || lookups != 1 || vault.deletes != 0 {
				t.Fatal("could not recover on the same device")
			}
			if persistent && (vault.value != next.RefreshToken || vault.saves != 1) {
				t.Fatal("new generation was not saved")
			}
		})
	}
}

func TestOtherRefreshFailuresStillDiscardCredentials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"malformed429", 429, `not json`},
		{"unrecognized429", 429, `{"error":{"code":"quota_exceeded"}}`},
		{"wrongStatus", 503, `{"error":{"code":"rate_limited"}}`},
		{"invalidToken", 401, `{"error":{"code":"invalid_token"}}`},
		{"revoked", 403, `{"error":{"code":"forbidden"}}`},
		{"lostRotationResult", 500, `{"error":{"code":"temporarily_unavailable"}}`},
		{"malformedSuccess", 200, `{`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := connected(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			c.credentials.AccessExpiresAt = time.Now().Add(-time.Minute)
			vault := &refreshTestVault{value: c.credentials.RefreshToken}
			c.persistent, c.vault = true, vault
			for i := 0; i < 2; i++ {
				if err := c.authorized(context.Background(), "GET", "session", nil, nil, ""); !errors.Is(err, ErrLogin) {
					t.Fatal(err)
				}
			}
			if calls != 1 || c.HasCredentials() || c.DeviceSessionID() != "" || vault.value != "" || vault.deletes != 1 {
				t.Fatal("uncertain or invalid credential retained or replayed")
			}
		})
	}
}
