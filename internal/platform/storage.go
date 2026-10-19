// Package platform contains user-local storage and Windows adapters.
package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// PreviewNamespace separates this offline preview from future product origins.
// Production origin must be fixed after the server-owned contract is available.
const PreviewNamespace = "llvy-uploader:offline-preview:v1"

type Config struct {
	ReplayFolder string `json:"replayFolder"`
}
type Store struct {
	dir  string
	lock io.Closer
}

func OpenStore(base, origin string) (*Store, error) {
	hash := sha256.Sum256([]byte(origin))
	dir := filepath.Join(base, "LLVY", "Uploader", hex.EncodeToString(hash[:]))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	lock, err := lockFile(filepath.Join(dir, "instance.lock"))
	if err != nil {
		return nil, err
	}
	return &Store{dir, lock}, nil
}
func (s *Store) Close() error { return s.lock.Close() }
func (s *Store) Load() (Config, error) {
	f, err := os.Open(filepath.Join(s.dir, "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	var c Config
	d := json.NewDecoder(io.LimitReader(f, 64<<10))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return Config{}, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return Config{}, errors.New("설정 형식 오류")
	}
	return c, nil
}
func (s *Store) Save(c Config) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return atomicWrite(s.dir, "config.json", data)
}

// Clear removes only app-owned data, preserving the held instance lock.
func (s *Store) Clear() error {
	for _, name := range []string{"config.json", "refresh.bin"} {
		if err := os.Remove(filepath.Join(s.dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
func atomicWrite(dir, name string, data []byte) error {
	f, err := os.CreateTemp(dir, ".write-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return replaceFile(f.Name(), filepath.Join(dir, name))
}
