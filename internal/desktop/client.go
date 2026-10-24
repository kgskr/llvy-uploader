package desktop

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var uuidPattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
var tokenPattern = regexp.MustCompile(`^(access|refresh)\.[A-Za-z0-9_-]{43}$`)
var codePattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

type Client struct {
	origin      string
	http        *http.Client
	mu          sync.Mutex
	credentials Credentials
	persistent  bool
	vault       Vault
}

func New(origin string, development bool, vault Vault) (*Client, error) {
	if err := ValidateOrigin(origin, development); err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	transport.ResponseHeaderTimeout = 20 * time.Second
	return &Client{origin: origin, vault: vault, http: &http.Client{Transport: transport, Timeout: 90 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) Origin() string { return c.origin }

// HasCredentials also covers restoration before the device identity is known.
// It does not establish current server authority.
func (c *Client) HasCredentials() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.credentials.RefreshToken != ""
}

// DeviceSessionID identifies locally retained credentials, not current server
// authority. Only Session and authorized operations verify that authority.
func (c *Client) DeviceSessionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.credentials.RefreshToken == "" {
		return ""
	}
	return c.credentials.DeviceSessionID
}
func (c *Client) do(ctx context.Context, method, path string, input, output any, credential, key string) error {
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return ErrContract
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.origin+"/api/desktop/v1/"+path, body)
	if err != nil {
		return ErrContract
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if credential != "" {
		req.Header.Set("Authorization", "Bearer "+credential)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	response, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrNetwork
	}
	defer response.Body.Close()
	b, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(b) > 64<<10 {
		return ErrNetwork
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var failure struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		code := "invalid_response"
		if json.Unmarshal(b, &failure) == nil && codePattern.MatchString(failure.Error.Code) {
			code = failure.Error.Code
		}
		retry := time.Duration(0)
		if n, e := strconv.Atoi(response.Header.Get("Retry-After")); e == nil && n >= 0 {
			retry = time.Duration(min(n, 86400)) * time.Second
		} else if t, e := http.ParseTime(response.Header.Get("Retry-After")); e == nil {
			retry = max(0, min(time.Until(t), 24*time.Hour))
		}
		return &APIError{Code: code, Status: response.StatusCode, RetryAfter: retry}
	}
	if output != nil && json.Unmarshal(b, output) != nil {
		return ErrContract
	}
	return nil
}
func validateCredentials(v Credentials) error {
	if v.Protocol != Protocol || v.TokenType != "Bearer" || !strings.HasPrefix(v.AccessToken, "access.") || !strings.HasPrefix(v.RefreshToken, "refresh.") || !tokenPattern.MatchString(v.AccessToken) || !tokenPattern.MatchString(v.RefreshToken) || !uuidPattern.MatchString(v.DeviceSessionID) || v.Scope != "replay:upload replay:status" || v.AccessExpiresAt.Before(time.Now()) || v.AccessExpiresAt.After(v.DeviceExpiresAt) || time.Until(v.AccessExpiresAt) > 16*time.Minute || time.Until(v.DeviceExpiresAt) > 31*24*time.Hour {
		return ErrContract
	}
	return nil
}
func (c *Client) accept(v Credentials, persistent bool) error {
	if err := validateCredentials(v); err != nil {
		return err
	}
	if persistent {
		if c.vault == nil {
			return ErrLogin
		}
		if err := c.vault.SaveRefresh(v.RefreshToken); err != nil {
			c.credentials = Credentials{}
			_ = c.vault.DeleteRefresh()
			return ErrLogin
		}
	}
	c.credentials = v
	c.persistent = persistent
	return nil
}
func (c *Client) clear() {
	c.credentials = Credentials{}
	c.persistent = false
	if c.vault != nil {
		_ = c.vault.DeleteRefresh()
	}
}

// A lost refresh response is terminal locally: replay would revoke the family.
func (c *Client) refreshLocked(ctx context.Context) error {
	old := c.credentials
	if old.RefreshToken == "" {
		return ErrLogin
	}
	var next Credentials
	if err := c.do(ctx, "POST", "auth/refresh", map[string]string{"refreshToken": old.RefreshToken}, &next, "", ""); err != nil {
		var api *APIError
		// This exact server rejection precedes rotation. All other failures
		// may hide a consumed token, so they must still require a new login.
		if errors.As(err, &api) && api.Status == http.StatusTooManyRequests && api.Code == "rate_limited" {
			return err
		}
		c.clear()
		return ErrLogin
	}
	if old.DeviceSessionID != "" && (next.DeviceSessionID != old.DeviceSessionID || !next.DeviceExpiresAt.Equal(old.DeviceExpiresAt)) {
		c.clear()
		return ErrContract
	}
	if err := c.accept(next, c.persistent); err != nil {
		c.clear()
		return err
	}
	return nil
}
func (c *Client) Restore(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.vault == nil {
		return nil
	}
	value, err := c.vault.LoadRefresh()
	if err != nil {
		c.clear()
		return ErrLogin
	}
	if value == "" {
		return nil
	}
	if !strings.HasPrefix(value, "refresh.") || !tokenPattern.MatchString(value) {
		c.clear()
		return ErrLogin
	}
	c.credentials = Credentials{RefreshToken: value}
	c.persistent = true
	return c.refreshLocked(ctx)
}
func (c *Client) authorized(ctx context.Context, method, path string, input, output any, key string) error {
	return c.authorizedFor(ctx, method, path, input, output, key, 30*time.Second)
}

func (c *Client) authorizedFor(ctx context.Context, method, path string, input, output any, key string, validity time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.credentials.RefreshToken == "" {
		return ErrLogin
	}
	refreshed := false
	if time.Until(c.credentials.AccessExpiresAt) < validity {
		if err := c.refreshLocked(ctx); err != nil {
			return err
		}
		refreshed = true
	}
	if time.Until(c.credentials.AccessExpiresAt) < validity {
		c.clear()
		return ErrLogin
	}
	err := c.do(ctx, method, path, input, output, c.credentials.AccessToken, key)
	var api *APIError
	if errors.As(err, &api) && api.Status == 401 && !refreshed {
		if e := c.refreshLocked(ctx); e != nil {
			return e
		}
		if time.Until(c.credentials.AccessExpiresAt) < validity {
			c.clear()
			return ErrLogin
		}
		return c.do(ctx, method, path, input, output, c.credentials.AccessToken, key)
	}
	return err
}
func (c *Client) Logout(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	value := c.credentials.RefreshToken
	var remote error
	if value != "" {
		remote = c.do(ctx, "POST", "auth/revoke", nil, nil, value, "")
	}
	c.credentials = Credentials{}
	c.persistent = false
	if c.vault != nil {
		if err := c.vault.DeleteRefresh(); err != nil {
			return errors.New("로컬 로그인 정보 삭제 실패")
		}
	}
	if remote != nil {
		return errors.New("로컬 로그아웃 완료 · 서버 연결 해제를 확인하지 못했습니다")
	}
	return nil
}
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	persistent := c.persistent
	c.mu.Unlock()
	if !persistent {
		return c.Logout(ctx)
	}
	c.mu.Lock()
	c.credentials = Credentials{}
	c.mu.Unlock()
	return nil
}
func (c *Client) Session(ctx context.Context) (Session, error) {
	var v Session
	err := c.authorized(ctx, "GET", "session", nil, &v, "")
	if err == nil && (v.Protocol != Protocol || !uuidPattern.MatchString(v.DeviceSessionID) || (v.Account.Role != "uploader" && v.Account.Role != "admin" && v.Account.Role != "owner") || v.MaxUploadBytes != 64<<20 || v.Quota.Account.Remaining < 0 || v.Quota.Global.Remaining < 0) {
		err = ErrContract
	}
	return v, err
}
func (c *Client) Reserve(ctx context.Context, m Metadata, key string) (Reservation, error) {
	var v Reservation
	err := c.safe(ctx, func() error { return c.authorized(ctx, "POST", "uploads", m, &v, key) })
	if err == nil && (!uuidPattern.MatchString(v.UploadID) || len(v.Nonce) != 43 || v.ExpiresAt.Before(time.Now())) {
		err = ErrContract
	}
	return v, err
}
func (c *Client) Status(ctx context.Context, id string) (Result, error) {
	var v Result
	if !uuidPattern.MatchString(id) {
		return v, ErrContract
	}
	err := c.safe(ctx, func() error { return c.authorized(ctx, "GET", "uploads/"+id, nil, &v, "") })
	if err == nil {
		err = validResult(id, v)
	}
	return v, err
}
func (c *Client) Process(ctx context.Context, b Reservation) (Result, error) {
	var v Result
	if !uuidPattern.MatchString(b.UploadID) {
		return v, ErrContract
	}
	// Cover the server's 60-second execution limit plus transport/clock margin.
	err := c.authorizedFor(ctx, "POST", "uploads/"+b.UploadID+"/process", map[string]string{"nonce": b.Nonce}, &v, "", 2*time.Minute)
	if err == nil {
		err = validResult(b.UploadID, v)
	}
	return v, err
}
func validResult(id string, r Result) error {
	if r.UploadID != id {
		return ErrContract
	}
	switch r.State {
	case "pending", "processing", "failed", "outcome_unknown":
		return nil
	case "completed", "duplicate":
		if uuidPattern.MatchString(r.GameID) && r.GamePath == "/games/"+r.GameID && r.Duplicate == (r.State == "duplicate") {
			return nil
		}
	}
	return ErrContract
}
func wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
func (c *Client) safe(ctx context.Context, run func() error) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = run()
		if err == nil {
			return nil
		}
		var api *APIError
		delay := time.Duration(attempt+1) * time.Second
		if errors.As(err, &api) {
			if api.Status < 500 {
				return err
			}
			delay = max(delay, api.RetryAfter)
		} else if !errors.Is(err, ErrNetwork) {
			return err
		}
		if attempt == 2 {
			return err
		}
		if delay > 30*time.Second {
			return err
		}
		if e := wait(ctx, delay); e != nil {
			return e
		}
	}
	return err
}
