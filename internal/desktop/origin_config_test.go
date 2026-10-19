package desktop

import "testing"

func TestBuildOriginHasNoDefault(t *testing.T) {
	t.Setenv("LLVY_SERVER_ORIGIN", "https://ignored.example.invalid")
	origin, _ := BuildOrigin()
	if origin != "" {
		t.Fatal("unconfigured test build has a default origin")
	}
	if _, err := New(origin, false, nil); err == nil {
		t.Fatal("unconfigured origin accepted")
	}
}

func TestConfiguredOrigin(t *testing.T) {
	for _, origin := range []string{"https://uploader.example.invalid", "https://staging.example.invalid:8443", "https://[::1]:443"} {
		if _, err := New(origin, false, nil); err != nil {
			t.Errorf("%s: %v", origin, err)
		}
	}
	for _, origin := range []string{"", "http://example.invalid", "http://127.0.0.1:3000", "https://example.invalid/", "https://example.invalid/path", "https://user:pass@example.invalid", "https://example.invalid?", "https://example.invalid#", "https://example.invalid?q=1", "https://example.invalid:0", "https://example.invalid:65536", "https://example.invalid:", "https://bad_host", "https://example.invalid -X injected=value", "https://example.invalid\n", "https://example.invalid'", "https://example.invalid$()"} {
		if err := ValidateOrigin(origin, false); err == nil {
			t.Errorf("accepted invalid origin %q", origin)
		}
	}
	if err := ValidateOrigin("http://127.0.0.1:3000", true); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOrigin("http://example.invalid", true); err == nil {
		t.Fatal("development allowed non-loopback HTTP")
	}
}
