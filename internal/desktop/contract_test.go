package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPinnedServerContract(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "desktop-v1")
	raw, err := os.ReadFile(filepath.Join(root, "source.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Commit   string            `json:"commit"`
		Protocol string            `json:"protocol"`
		SHA256   map[string]string `json:"sha256"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	if len(source.Commit) != 40 || source.Protocol != Protocol {
		t.Fatal("invalid contract provenance")
	}
	for name, expected := range source.SHA256 {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != expected {
			t.Fatalf("source checksum mismatch: %s", name)
		}
	}
	raw, _ = os.ReadFile(filepath.Join(root, "protocol.json"))
	var protocol struct {
		ClientID      string `json:"clientId"`
		ProductOrigin string `json:"productOrigin"`
	}
	if err := json.Unmarshal(raw, &protocol); err != nil {
		t.Fatal(err)
	}
	if protocol.ClientID != ClientID || protocol.ProductOrigin != ProductOrigin {
		t.Fatal("server identity mismatch")
	}
	raw, _ = os.ReadFile(filepath.Join(root, "fixtures.json"))
	var fixtures map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	var session Session
	if err := json.Unmarshal(fixtures["session"], &session); err != nil {
		t.Fatal(err)
	}
	if session.Protocol != Protocol || session.Quota.Account.Remaining != 9 || session.MaxUploadBytes != 64<<20 {
		t.Fatal("session mismatch")
	}
	for _, name := range []string{"completed", "duplicate", "processing", "unknown"} {
		var result Result
		if err := json.Unmarshal(fixtures[name], &result); err != nil {
			t.Fatal(err)
		}
		if err := validResult(fixtureID, result); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	var transfer Transfer
	if err := json.Unmarshal(fixtures["transfer"], &transfer); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTransfer(transfer, fixtureID, transfer.ExpiresAt.Add(-4*time.Minute)); err != nil {
		t.Fatal(err)
	}
}
