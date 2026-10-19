//go:build windows

package platform

import (
	"bytes"
	"testing"
)

func TestDPAPIUserRoundTrip(t *testing.T) {
	plain := []byte("synthetic test credential")
	encrypted, err := Protect(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(plain, encrypted) {
		t.Fatal("plaintext stored")
	}
	got, err := Unprotect(encrypted)
	if err != nil || !bytes.Equal(plain, got) {
		t.Fatalf("DPAPI failed: %v", err)
	}
	if _, err := Unprotect([]byte("invalid blob")); err == nil {
		t.Fatal("corruption accepted")
	}
}
func TestKnownFolders(t *testing.T) {
	if path, err := LocalData(); err != nil || path == "" {
		t.Fatal(err)
	}
	if _, err := Candidates(); err != nil {
		t.Fatal(err)
	}
}
