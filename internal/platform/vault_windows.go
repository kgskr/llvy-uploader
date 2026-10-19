package platform

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

func (s *Store) LoadRefresh() (string, error) {
	f, err := os.Open(filepath.Join(s.dir, "refresh.bin"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (16<<10)+1))
	if err != nil || len(data) > 16<<10 {
		return "", errors.New("로그인 정보 형식 오류")
	}
	plain, err := Unprotect(data)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
func (s *Store) SaveRefresh(token string) error {
	data, err := Protect([]byte(token))
	if err != nil {
		return err
	}
	return atomicWrite(s.dir, "refresh.bin", data)
}
func (s *Store) DeleteRefresh() error {
	err := os.Remove(filepath.Join(s.dir, "refresh.bin"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
