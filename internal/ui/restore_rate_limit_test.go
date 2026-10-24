package ui

import (
	"context"
	"encoding/json"
	"errors"
	"llvy-uploader/internal/desktop"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type restoreTestVault struct {
	value   string
	deletes int
}

func (v *restoreTestVault) LoadRefresh() (string, error)   { return v.value, nil }
func (v *restoreTestVault) SaveRefresh(value string) error { v.value = value; return nil }
func (v *restoreTestVault) DeleteRefresh() error           { v.value = ""; v.deletes++; return nil }

func TestThrottledStartupRetriesSavedConnectionWithoutBrowser(t *testing.T) {
	oldToken := "refresh." + strings.Repeat("A", 43)
	vault := &restoreTestVault{value: oldToken}
	refreshes, lookups, browsers := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		write := func(v any) { _ = json.NewEncoder(w).Encode(v) }
		switch r.URL.Path {
		case "/api/desktop/v1/auth/refresh":
			refreshes++
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["refreshToken"] != oldToken {
				t.Error("saved token was not reused")
			}
			if refreshes == 1 {
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(429)
				write(map[string]any{"error": map[string]string{"code": "rate_limited"}})
				return
			}
			write(desktop.Credentials{Protocol: desktop.Protocol, TokenType: "Bearer", AccessToken: "access." + strings.Repeat("A", 43), RefreshToken: "refresh." + strings.Repeat("B", 43), DeviceSessionID: connectionID, Scope: "replay:upload replay:status", AccessExpiresAt: time.Now().Add(10 * time.Minute), DeviceExpiresAt: time.Now().Add(time.Hour)})
		case "/api/desktop/v1/session":
			lookups++
			s := desktop.Session{Protocol: desktop.Protocol, DeviceSessionID: connectionID, MaxUploadBytes: 64 << 20}
			s.Account.Role = "uploader"
			write(s)
		default:
			t.Errorf("unexpected new authorization: %s", r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	client, err := desktop.New(server.URL, true, vault)
	if err != nil {
		t.Fatal(err)
	}
	first := runClientWork(context.Background(), client, func(ctx context.Context) update {
		_, err := restoreSession(ctx, client)
		var api *desktop.APIError
		if !errors.As(err, &api) || api.Status != 429 || api.RetryAfter != time.Minute {
			t.Fatalf("lost throttling result: %v", err)
		}
		return update{message: err.Error(), keepRows: true}
	})
	if refreshes != 1 || lookups != 0 || !first.hasCredentials || first.signedOut || first.deviceID != "" || vault.deletes != 0 || vault.value != oldToken {
		t.Fatal("startup replayed or discarded the saved connection")
	}
	second := runClientWork(context.Background(), client, func(ctx context.Context) update {
		return connect(ctx, client, false, func(string) error { browsers++; return errors.New("must not open browser") })
	})
	if second.session == nil || second.deviceID != connectionID || !second.hasCredentials || second.signedOut || refreshes != 2 || lookups != 1 || browsers != 0 || vault.deletes != 0 {
		t.Fatalf("did not resume restored connection: %+v", second)
	}
	if vault.value == oldToken {
		t.Fatal("rotated token was not persisted")
	}
}
