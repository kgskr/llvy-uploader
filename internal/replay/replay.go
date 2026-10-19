// Package replay holds a stable read handle from validation through streaming.
package replay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"llvy-uploader/internal/discovery"
	"os"
)

var ErrChanged = errors.New("파일이 변경되었습니다. 다시 찾아 선택하세요")

type Handle struct {
	*os.File
	info os.FileInfo
}

func Open(expected discovery.File) (*Handle, error) {
	if expected.Problem != "" {
		return nil, errors.New(expected.Problem)
	}
	current, include, err := (discovery.OSReader{}).Inspect(expected.Path)
	if err != nil {
		return nil, err
	}
	if !include || current.Size != expected.Size || !current.ModTime.Equal(expected.ModTime) {
		return nil, ErrChanged
	}
	if current.Problem != "" {
		return nil, errors.New(current.Problem)
	}
	f, err := openRead(expected.Path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !regular(info) || info.Size() < 1 || info.Size() > discovery.MaxSize || info.Size() != expected.Size || !info.ModTime().Equal(expected.ModTime) {
		f.Close()
		return nil, ErrChanged
	}
	if err := checkHandle(f); err != nil {
		f.Close()
		return nil, err
	}
	return &Handle{f, info}, nil
}
func (h *Handle) Check() error {
	info, err := h.Stat()
	if err != nil {
		return err
	}
	if info.Size() != h.info.Size() || !info.ModTime().Equal(h.info.ModTime()) {
		return ErrChanged
	}
	return checkHandle(h.File)
}
func (h *Handle) Hash(ctx context.Context) (string, error) {
	if _, err := h.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hash := sha256.New()
	buffer := make([]byte, 64<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := h.Read(buffer)
		total += int64(n)
		if total > h.info.Size() {
			return "", ErrChanged
		}
		if n > 0 {
			_, _ = hash.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	if total != h.info.Size() {
		return "", ErrChanged
	}
	if err := h.Check(); err != nil {
		return "", err
	}
	if _, err := h.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
