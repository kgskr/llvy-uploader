package desktop

import (
	"context"
	"errors"
	"io"
	desktopcontract "llvy-uploader/contracts/desktop-v1"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func ValidateTransfer(v Transfer, id string, now time.Time) error {
	origin, path, method := desktopcontract.Endpoint()
	u, err := url.Parse(v.UploadURL)
	if err != nil || u.Scheme != "https" || u.Scheme+"://"+u.Host != origin || u.User != nil || u.Path != path || u.RawPath != "" || u.Fragment != "" || v.Method != method || !uuidPattern.MatchString(id) || v.ExpiresAt.Before(now) || v.ExpiresAt.After(now.Add(5*time.Minute+5*time.Second)) {
		return ErrContract
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q["pathname"]) != 1 || q.Get("pathname") != "replays/"+id+".rofl" {
		return ErrContract
	}
	seen := map[string]bool{}
	for key, value := range v.Headers {
		name := strings.ToLower(key)
		if seen[name] || strings.ContainsAny(value, "\r\n") {
			return ErrContract
		}
		seen[name] = true
		if !desktopcontract.HeaderAllowed(name, value) {
			return ErrContract
		}
	}
	if len(seen) != desktopcontract.HeaderCount() {
		return ErrContract
	}
	return nil
}
func (c *Client) Transfer(ctx context.Context, b Reservation) (Transfer, error) {
	var v Transfer
	if !uuidPattern.MatchString(b.UploadID) {
		return v, ErrContract
	}
	err := c.safe(ctx, func() error {
		return c.authorized(ctx, "POST", "uploads/"+b.UploadID+"/transfer", map[string]string{"nonce": b.Nonce}, &v, "")
	})
	if err == nil {
		err = ValidateTransfer(v, b.UploadID, time.Now())
	}
	return v, err
}

type progressReader struct {
	reader      io.Reader
	read, total int64
	progress    func(int64, int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	p.read += int64(n)
	if p.progress != nil {
		p.progress(p.read, p.total)
	}
	return n, err
}

// PUT has no automatic retries and never contains the LLVY bearer credential.
func (c *Client) Put(ctx context.Context, v Transfer, id string, file io.Reader, size int64, progress func(int64, int64)) error {
	if err := ValidateTransfer(v, id, time.Now()); err != nil {
		return err
	}
	if size < 1 || size > 64<<20 {
		return ErrContract
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", v.UploadURL, &progressReader{reader: io.LimitReader(file, size), total: size, progress: progress})
	if err != nil {
		return ErrContract
	}
	req.ContentLength = size
	for key, value := range v.Headers {
		req.Header.Set(key, value)
	}
	response, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrNetwork
	}
	defer response.Body.Close()
	// Object URL returned by Blob is deliberately not trusted or forwarded.
	_, err = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	if err != nil {
		return ErrNetwork
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New("전송 응답을 확인해야 합니다")
	}
	return nil
}
