package ui

import (
	"context"
	"encoding/json"
	"llvy-uploader/internal/app"
	"llvy-uploader/internal/desktop"
	"llvy-uploader/internal/discovery"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const connectionID = "11111111-1111-4111-8111-111111111111"

func TestConnectionLookupCanResumeAfterLogin(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "server failure", true: "cancellation"}[canceled], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var redirect, state string
			requests, exchanges, lookups, browsers := 0, 0, 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				write := func(v any) { _ = json.NewEncoder(w).Encode(v) }
				switch r.URL.Path {
				case "/api/desktop/v1/auth/requests":
					requests++
					var b map[string]string
					_ = json.NewDecoder(r.Body).Decode(&b)
					redirect, state = b["redirectUri"], b["state"]
					write(map[string]any{"protocol": desktop.Protocol, "requestId": connectionID, "approvalPath": "/desktop/connect?request=" + connectionID, "expiresAt": time.Now().Add(time.Minute)})
				case "/api/desktop/v1/auth/token":
					exchanges++
					write(desktop.Credentials{Protocol: desktop.Protocol, TokenType: "Bearer", AccessToken: "access." + strings.Repeat("A", 43), RefreshToken: "refresh." + strings.Repeat("B", 43), DeviceSessionID: connectionID, Scope: "replay:upload replay:status", AccessExpiresAt: time.Now().Add(10 * time.Minute), DeviceExpiresAt: time.Now().Add(time.Hour)})
				case "/api/desktop/v1/session":
					lookups++
					if lookups == 1 {
						if canceled {
							cancel()
							<-r.Context().Done()
							return
						}
						w.WriteHeader(503)
						write(map[string]any{"error": map[string]string{"code": "temporarily_unavailable"}})
						return
					}
					s := desktop.Session{Protocol: desktop.Protocol, DeviceSessionID: connectionID, MaxUploadBytes: 64 << 20}
					s.Account.Role = "owner"
					write(s)
				case "/api/desktop/v1/auth/revoke":
					write(map[string]bool{"ok": true})
				default:
					t.Errorf("unexpected route %s", r.URL.Path)
				}
			}))
			defer server.Close()
			client, err := desktop.New(server.URL, true, nil)
			if err != nil {
				t.Fatal(err)
			}
			open := func(string) error {
				browsers++
				r, err := http.Get(redirect + "?state=" + state + "&code=code." + strings.Repeat("C", 43))
				if err == nil {
					r.Body.Close()
				}
				return err
			}
			first := runClientWork(ctx, client, func(ctx context.Context) update { return connect(ctx, client, false, open) })
			if first.session != nil || first.deviceID != connectionID || first.signedOut || !strings.Contains(first.message, "연결 정보가 남아") {
				t.Fatalf("lost connection state: %+v", first)
			}
			second := runClientWork(context.Background(), client, func(ctx context.Context) update { return connect(ctx, client, false, open) })
			if second.session == nil || second.deviceID != connectionID || requests != 1 || exchanges != 1 || browsers != 1 || lookups != 2 {
				t.Fatalf("did not resume same connection: %+v counts %d %d %d %d", second, requests, exchanges, browsers, lookups)
			}
			// A worker must not revive the old snapshot after local logout.
			ended := runClientWork(context.Background(), client, func(ctx context.Context) update { _ = client.Logout(ctx); return update{session: second.session} })
			if !ended.signedOut || ended.deviceID != "" || ended.session != nil {
				t.Fatalf("stale snapshot revived login: %+v", ended)
			}
		})
	}
}

func TestDeviceChangeRequiresExplicitNewSelection(t *testing.T) {
	for _, known := range []bool{false, true} {
		e := desktop.NewEntry(discovery.File{Path: "replay.rofl"})
		e.ReservationAttempted = true
		e.DeviceSessionID = connectionID
		if known {
			e.Binding.UploadID = connectionID
		}
		entries := map[string]*desktop.Entry{e.File.Path: e}
		for _, id := range []string{"", "22222222-2222-4222-8222-222222222222"} {
			result := reconcileEntries(entries, update{authChecked: true, deviceID: id, keepRows: true, rows: []app.Row{{}}})
			if !result.clearEntries || result.keepRows || result.rows != nil || !strings.Contains(result.message, "예약 한도") {
				t.Fatalf("silent cross-device recovery: %+v", result)
			}
		}
		result := reconcileEntries(entries, update{authChecked: true, deviceID: connectionID, keepRows: true})
		if result.clearEntries || !result.keepRows {
			t.Fatal("same-device refresh discarded recovery")
		}
	}
}

func TestCanceledDiscoveryAndSearchDoNotReplaceSelection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := runWork(ctx, func(ctx context.Context) update {
		found := discovery.Find(ctx, discovery.OSReader{}, "", []discovery.Candidate{{Path: t.TempDir(), Source: "Windows 문서"}})
		return update{selectionResult: true, folders: found.Candidates, discovered: true, picker: len(found.Candidates) == 0}
	})
	if result.picker || result.discovered || !result.keepRows || result.folders != nil {
		t.Fatalf("canceled discovery navigates: %+v", result)
	}
	result = runWork(ctx, func(context.Context) update { return update{selectionResult: true, rows: []app.Row{{}}} })
	if result.rows != nil || !result.keepRows {
		t.Fatal("canceled search replaced rows")
	}
	// Committed explicit folder selection is independent of automatic discovery.
	result = runWork(ctx, func(context.Context) update {
		return update{discovered: true, folders: []discovery.Candidate{{Path: "chosen"}}}
	})
	if !result.discovered || len(result.folders) != 1 {
		t.Fatal("lost committed folder change")
	}
}
