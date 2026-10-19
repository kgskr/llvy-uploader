//go:build darwin || linux

package platform

import (
	"errors"
	"golang.org/x/sys/unix"
	"io"
	"os"
)

func lockFile(path string) (io.Closer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("이미 실행 중입니다")
	}
	return f, nil
}
func replaceFile(from, to string) error { return os.Rename(from, to) }
