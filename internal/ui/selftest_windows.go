package ui

import (
	"bytes"
	"errors"
	"llvy-uploader/internal/platform"
	"os"
	"time"
)

// SelfTest is bounded, uses only temporary synthetic data and performs no network requests.
func SelfTest() error {
	done := make(chan error, 1)
	go func() { done <- selfTest() }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		return errors.New("self-test timeout")
	}
}
func selfTest() error {
	dir, err := os.MkdirTemp("", "llvy-self-test-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	s, err := platform.OpenStore(dir, "self-test")
	if err != nil {
		return err
	}
	defer s.Close()
	if other, err := platform.OpenStore(dir, "self-test"); err == nil {
		other.Close()
		return errors.New("exclusive lock failed")
	}
	if err := s.Save(platform.Config{ReplayFolder: "한글 경로"}); err != nil {
		return err
	}
	c, err := s.Load()
	if err != nil {
		return err
	}
	if c.ReplayFolder != "한글 경로" {
		return errors.New("unicode config failed")
	}
	secret := []byte("synthetic self-test credential")
	encrypted, err := platform.Protect(secret)
	if err != nil {
		return err
	}
	plain, err := platform.Unprotect(encrypted)
	if err != nil {
		return err
	}
	if !bytes.Equal(secret, plain) {
		return errors.New("DPAPI round trip failed")
	}
	_, err = platform.LocalData()
	return err
}
