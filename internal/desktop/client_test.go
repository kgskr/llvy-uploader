package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"llvy-uploader/internal/discovery"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testOrigin = "https://uploader.example.invalid"

const fixtureID = "11111111-1111-4111-8111-111111111111"

func credentials() Credentials {
	return Credentials{Protocol: Protocol, TokenType: "Bearer", AccessToken: "access." + strings.Repeat("A", 43), RefreshToken: "refresh." + strings.Repeat("B", 43), AccessExpiresAt: time.Now().Add(14 * time.Minute).UTC(), DeviceExpiresAt: time.Now().Add(12 * time.Hour).UTC(), DeviceSessionID: fixtureID, Scope: "replay:upload replay:status"}
}
func connected(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := New(server.URL, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	c.credentials = credentials()
	return c
}
func jsonResponse(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
func TestOriginAndRedaction(t *testing.T) {
	for _, u := range []string{"http://llvy.my", "https://llvy.my/", "https://u:p@llvy.my", "https://llvy.my?token=secret"} {
		if _, err := New(u, false, nil); err == nil {
			t.Errorf("allowed %s", u)
		}
	}
	c := connected(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		jsonResponse(w, map[string]any{"error": map[string]string{"code": "https://secret.invalid/?token=PRIVATE"}})
	})
	err := c.do(context.Background(), "GET", "session", nil, nil, "", "")
	if strings.Contains(err.Error(), "PRIVATE") {
		t.Fatal("leaked server data")
	}
}
func TestRedirectAndResponseLimit(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected.Add(1) }))
	defer target.Close()
	c := connected(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) })
	if err := c.do(context.Background(), "GET", "session", nil, nil, "credential", ""); err == nil {
		t.Fatal("accepted redirect")
	}
	if redirected.Load() != 0 {
		t.Fatal("followed redirect")
	}
	c = connected(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, strings.Repeat("x", 70<<10)) })
	if err := c.do(context.Background(), "GET", "session", nil, nil, "", ""); !errors.Is(err, ErrNetwork) {
		t.Fatal(err)
	}
}
func TestSerializedRefreshAndUncertainResponse(t *testing.T) {
	var calls atomic.Int32
	next := credentials()
	c := connected(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "refresh") {
			calls.Add(1)
			jsonResponse(w, next)
			return
		}
		jsonResponse(w, map[string]string{"ok": "yes"})
	})
	c.credentials.AccessExpiresAt = time.Now().Add(-time.Minute)
	c.credentials.DeviceExpiresAt = next.DeviceExpiresAt
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.authorized(context.Background(), "GET", "session", nil, nil, ""); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
	c = connected(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(503) })
	c.credentials.AccessExpiresAt = time.Now().Add(-time.Minute)
	before := calls.Load()
	for i := 0; i < 2; i++ {
		if err := c.authorized(context.Background(), "GET", "session", nil, nil, ""); !errors.Is(err, ErrLogin) {
			t.Fatal(err)
		}
	}
	if calls.Load() != before+1 {
		t.Fatal("refresh replayed")
	}
}
func TestLoginLoopbackStateAndPKCE(t *testing.T) {
	var start struct{ RedirectURI, State, CodeChallenge string }
	var next = credentials()
	var exchanged int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/desktop/v1/auth/requests":
			var b map[string]string
			_ = json.NewDecoder(r.Body).Decode(&b)
			start.RedirectURI = b["redirectUri"]
			start.State = b["state"]
			start.CodeChallenge = b["codeChallenge"]
			if b["mode"] != "transient" || b["codeChallengeMethod"] != "S256" {
				t.Error(b)
			}
			jsonResponse(w, map[string]any{"protocol": Protocol, "requestId": fixtureID, "approvalPath": "/desktop/connect?request=" + fixtureID, "expiresAt": time.Now().Add(10 * time.Minute)})
		case "/api/desktop/v1/auth/token":
			exchanged++
			var b map[string]string
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b["redirectUri"] != start.RedirectURI || len(b["codeVerifier"]) != 43 || b["clientId"] != ClientID {
				t.Error(b)
			}
			jsonResponse(w, next)
		default:
			t.Error(r.URL.Path)
		}
	}))
	defer server.Close()
	c, _ := New(server.URL, true, nil)
	err := c.Login(context.Background(), false, func(target string) error {
		if target != server.URL+"/desktop/connect?request="+fixtureID {
			t.Fatal(target)
		}
		for _, query := range []string{"state=wrong&code=code." + strings.Repeat("C", 43), "state=" + start.State + "&state=" + start.State + "&code=code." + strings.Repeat("C", 43)} {
			r, e := http.Get(start.RedirectURI + "?" + query)
			if e != nil {
				return e
			}
			r.Body.Close()
			if r.StatusCode != 400 {
				t.Fatal(r.StatusCode)
			}
		}
		r, e := http.Get(start.RedirectURI + "?state=" + start.State + "&code=code." + strings.Repeat("C", 43))
		if e == nil {
			r.Body.Close()
		}
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	if exchanged != 1 {
		t.Fatal(exchanged)
	}
	if _, err := http.Get(start.RedirectURI); err == nil {
		t.Fatal("listener not closed")
	}
}
func descriptor() Transfer {
	return Transfer{UploadURL: "https://vercel.com/api/blob/?pathname=replays%2F" + fixtureID + ".rofl&vercel-blob-signature=fixture", Method: "PUT", Headers: map[string]string{"content-type": "application/octet-stream", "x-content-type": "application/octet-stream", "x-api-version": "12", "x-vercel-blob-access": "private", "x-vercel-blob-store-id": "fixturestore"}, ExpiresAt: time.Now().Add(4 * time.Minute)}
}
func TestDescriptorAllowlist(t *testing.T) {
	good := descriptor()
	if err := ValidateTransfer(good, fixtureID, time.Now()); err != nil {
		t.Fatal(err)
	}
	tests := []func(*Transfer){func(v *Transfer) { v.Headers["Authorization"] = "secret" }, func(v *Transfer) { v.Headers["Content-Type"] = "application/octet-stream" }, func(v *Transfer) { v.Headers["x-api-version"] = "12\r\nHost:evil" }, func(v *Transfer) {
		v.UploadURL = "https://evil.invalid/api/blob/?pathname=replays%2F" + fixtureID + ".rofl"
	}, func(v *Transfer) { v.UploadURL += "&pathname=other" }, func(v *Transfer) { v.Method = "POST" }, func(v *Transfer) { v.ExpiresAt = time.Now().Add(10 * time.Minute) }}
	for i, mutate := range tests {
		v := descriptor()
		mutate(&v)
		if ValidateTransfer(v, fixtureID, time.Now()) == nil {
			t.Errorf("accepted %d", i)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fileFixture(t *testing.T) discovery.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "KR_123.rofl")
	if err := os.WriteFile(path, []byte("fixture replay bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	return discovery.File{Path: path, Name: info.Name(), Size: info.Size(), ModTime: info.ModTime()}
}
func TestLostPUTAndProcessRecoverWithoutOverwrite(t *testing.T) {
	var reserved, put, processed, status int
	binding := Reservation{UploadID: fixtureID, Nonce: strings.Repeat("A", 43), ExpiresAt: time.Now().Add(30 * time.Minute)}
	result := Result{UploadID: fixtureID, State: "completed", GameID: fixtureID, GamePath: "/games/" + fixtureID}
	c := connected(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/desktop/v1/uploads":
			reserved++
			if r.Header.Get("Idempotency-Key") == "" {
				t.Error("no key")
			}
			jsonResponse(w, binding)
		case strings.HasSuffix(r.URL.Path, "/transfer"):
			jsonResponse(w, descriptor())
		case strings.HasSuffix(r.URL.Path, "/process"):
			processed++
			w.WriteHeader(503)
			jsonResponse(w, map[string]any{"error": map[string]string{"code": "outcome_unknown"}})
		default:
			status++
			jsonResponse(w, result)
		}
	})
	original := c.http.Transport
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "vercel.com" {
			put++
			if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.ContentLength != int64(len("fixture replay bytes")) {
				t.Error("unsafe PUT")
			}
			_, _ = io.ReadAll(r.Body)
			return nil, errors.New("response lost with signed secret URL")
		}
		return original.RoundTrip(r)
	})
	entry := NewEntry(fileFixture(t))
	if err := c.Upload(context.Background(), entry, nil); err != nil {
		t.Fatal(err)
	}
	if entry.Result.State != "completed" || reserved != 1 || put != 1 || processed != 1 || status != 1 {
		t.Fatalf("%+v %d %d %d %d", entry.Result, reserved, put, processed, status)
	}
	if err := c.Upload(context.Background(), entry, nil); err != nil {
		t.Fatal(err)
	}
	if put != 1 || reserved != 1 {
		t.Fatal("replayed")
	}
}
func TestReservationRetryKeepsKeyAndQuotaPauses(t *testing.T) {
	keys := []string{}
	c := connected(t, func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		if len(keys) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(503)
			jsonResponse(w, map[string]any{"error": map[string]string{"code": "temporarily_unavailable"}})
			return
		}
		w.WriteHeader(429)
		jsonResponse(w, map[string]any{"error": map[string]string{"code": "quota_exhausted"}})
	})
	_, err := c.Reserve(context.Background(), Metadata{}, "same-key")
	if !PauseQueue(err) || len(keys) != 2 || keys[0] != keys[1] {
		t.Fatalf("%v %v", err, keys)
	}
}
func TestCallbackCancellation(t *testing.T) {
	var callback string
	c := connected(t, func(w http.ResponseWriter, r *http.Request) {
		var b map[string]string
		_ = json.NewDecoder(r.Body).Decode(&b)
		callback = b["redirectUri"]
		jsonResponse(w, map[string]any{"protocol": Protocol, "requestId": fixtureID, "approvalPath": "/desktop/connect?request=" + fixtureID, "expiresAt": time.Now().Add(time.Minute)})
	})
	c.credentials = Credentials{}
	ctx, cancel := context.WithCancel(context.Background())
	err := c.Login(ctx, false, func(string) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	u, _ := url.Parse(callback)
	if u.Host == "" {
		t.Fatal("callback missing")
	}
	if _, err := http.Get(callback); err == nil {
		t.Fatal("listener leaked")
	}
}

func TestDuplicateJSONHeaderNames(t *testing.T) {
	for _, raw := range []string{`{"headers":{"x-api-version":"12","x-api-version":"12"}}`, `{"headers":{"Content-Type":"application/octet-stream","content-type":"application/octet-stream"}}`} {
		var value Transfer
		if json.Unmarshal([]byte(raw), &value) == nil {
			t.Fatal("duplicate JSON header accepted")
		}
	}
}

func TestSessionRoles(t *testing.T) {
	for _, role := range []string{"uploader", "admin", "owner", "viewer", "unknown", ""} {
		t.Run(role, func(t *testing.T) {
			c := connected(t, func(w http.ResponseWriter, r *http.Request) {
				v := Session{Protocol: Protocol, DeviceSessionID: fixtureID, MaxUploadBytes: 64 << 20}
				v.Account.Role = role
				jsonResponse(w, v)
			})
			got, err := c.Session(context.Background())
			allowed := role == "uploader" || role == "admin" || role == "owner"
			if allowed && (err != nil || got.Account.Role != role) {
				t.Fatalf("valid role %s: %v", role, err)
			}
			if !allowed && !errors.Is(err, ErrContract) {
				t.Fatalf("accepted role %s: %v", role, err)
			}
		})
	}
}
