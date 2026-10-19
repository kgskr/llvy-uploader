package desktop

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

func Random() string { return base64.RawURLEncoding.EncodeToString(randomBytes()) }
func randomBytes() []byte {
	v := make([]byte, 32)
	if _, err := rand.Read(v); err != nil {
		panic("secure random unavailable")
	}
	return v
}
func (c *Client) Login(parent context.Context, persistent bool, open func(string) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.credentials.RefreshToken != "" {
		return ErrContract
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	state, verifier := Random(), Random()
	digest := sha256.Sum256([]byte(verifier))
	redirect := "http://" + listener.Addr().String() + "/callback"
	mode := "transient"
	if persistent {
		mode = "persistent"
	}
	var start struct {
		Protocol     string    `json:"protocol"`
		RequestID    string    `json:"requestId"`
		ApprovalPath string    `json:"approvalPath"`
		ExpiresAt    time.Time `json:"expiresAt"`
	}
	err = c.do(ctx, "POST", "auth/requests", map[string]string{"clientId": ClientID, "redirectUri": redirect, "state": state, "codeChallenge": base64.RawURLEncoding.EncodeToString(digest[:]), "codeChallengeMethod": "S256", "mode": mode}, &start, "", "")
	if err != nil {
		return err
	}
	if start.Protocol != Protocol || !uuidPattern.MatchString(start.RequestID) || start.ApprovalPath != "/desktop/connect?request="+start.RequestID || start.ExpiresAt.Before(time.Now()) {
		return ErrContract
	}
	codes := make(chan string, 1)
	var once sync.Once
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		q, e := url.ParseQuery(r.URL.RawQuery)
		if r.Method != "GET" || r.Host != listener.Addr().String() || r.URL.Path != "/callback" || r.URL.RawPath != "" || len(r.URL.RawQuery) > 512 || e != nil || len(q) != 2 || len(q["state"]) != 1 || len(q["code"]) != 1 || q.Get("state") != state || !strings.HasPrefix(q.Get("code"), "code.") || len(q.Get("code")) != 48 {
			http.Error(w, "잘못된 연결 요청", 400)
			return
		}
		accepted := false
		once.Do(func() { codes <- q.Get("code"); accepted = true })
		if !accepted {
			http.Error(w, "이미 처리된 요청", 409)
			return
		}
		_, _ = w.Write([]byte("연결 승인을 받았습니다. 업로더 창으로 돌아가세요."))
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 5 * time.Second, MaxHeaderBytes: 8192}
	defer server.Close()
	done := make(chan struct{})
	go func() { defer close(done); _ = server.Serve(listener) }()
	defer func() { server.Close(); <-done }()
	if err := open(c.origin + start.ApprovalPath); err != nil {
		return err
	}
	var code string
	select {
	case code = <-codes:
	case <-ctx.Done():
		return ctx.Err()
	}
	var next Credentials
	if err := c.do(ctx, "POST", "auth/token", map[string]string{"clientId": ClientID, "redirectUri": redirect, "code": code, "codeVerifier": verifier}, &next, "", ""); err != nil {
		return err
	}
	if err := c.accept(next, persistent); err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = c.do(cleanup, "POST", "auth/revoke", nil, nil, next.RefreshToken, "")
		c.clear()
		return err
	}
	return nil
}
