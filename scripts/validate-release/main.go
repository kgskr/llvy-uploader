// Command validate-release checks reviewed release evidence without network access.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"llvy-uploader/internal/desktop"
)

type check struct {
	Status      string `json:"status"`
	EvidenceURL string `json:"evidenceUrl"`
}
type evidence struct {
	SchemaVersion  int               `json:"schemaVersion"`
	ClientCommit   string            `json:"clientCommit"`
	Version        string            `json:"version"`
	Origin         string            `json:"origin"`
	ContractCommit string            `json:"contractCommit"`
	ContractSHA256 map[string]string `json:"contractSha256"`
	ServerCommit   string            `json:"serverCommit"`
	DeploymentURL  string            `json:"deploymentUrl"`
	Checks         map[string]check  `json:"checks"`
}
type source struct {
	Commit string            `json:"commit"`
	SHA256 map[string]string `json:"sha256"`
}

var requiredChecks = []string{"serverCheck", "serverOpenSpec", "postgresAuthorizationRace", "windows10", "windows11", "discordLogin", "blobPublicPrivate", "replayDuplicateBrowser", "revocation"}
var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func evidenceURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(raw, "#") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host != "localhost" && host != "example.com" && host != "example.org" && !strings.HasSuffix(host, ".invalid") && !strings.HasSuffix(host, ".example") && !strings.HasSuffix(host, ".test")
}

func validate(raw []byte, version, commit, origin string, pinned source) (evidence, error) {
	var e evidence
	if len(raw) == 0 || len(raw) > 64*1024 {
		return e, errors.New("release evidence is missing or too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&e); err != nil {
		return e, errors.New("invalid release evidence JSON")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return e, errors.New("unexpected trailing evidence")
	}
	if err := desktop.ValidateOrigin(origin, false); err != nil {
		return e, errors.New("invalid release origin")
	}
	if e.SchemaVersion != 1 || !commitPattern.MatchString(commit) || e.ClientCommit != commit || version == "" || e.Version != version || e.Origin != origin {
		return e, errors.New("evidence does not match release version, source commit or origin")
	}
	if !commitPattern.MatchString(e.ServerCommit) || e.ServerCommit == strings.Repeat("0", 40) || !evidenceURL(e.DeploymentURL) {
		return e, errors.New("verified server deployment is required")
	}
	if !commitPattern.MatchString(pinned.Commit) || e.ContractCommit != pinned.Commit || len(e.ContractSHA256) != 3 || len(pinned.SHA256) != 3 {
		return e, errors.New("contract revision mismatch")
	}
	for _, name := range []string{"protocol.json", "schema.json", "fixtures.json"} {
		if len(pinned.SHA256[name]) != 64 || e.ContractSHA256[name] != pinned.SHA256[name] {
			return e, errors.New("contract checksum mismatch")
		}
	}
	if len(e.Checks) != len(requiredChecks) {
		return e, errors.New("all acceptance checks are required")
	}
	for _, name := range requiredChecks {
		c := e.Checks[name]
		if c.Status != "passed" || !evidenceURL(c.EvidenceURL) {
			return e, fmt.Errorf("passing acceptance evidence required: %s", name)
		}
	}
	return e, nil
}

func run() error {
	raw := []byte(os.Getenv("LLVY_RELEASE_EVIDENCE_JSON"))
	if len(os.Args) > 2 {
		return errors.New("usage: validate-release [evidence.json]")
	}
	if len(os.Args) == 2 {
		var err error
		raw, err = os.ReadFile(os.Args[1])
		if err != nil {
			return errors.New("cannot read release evidence")
		}
	}
	data, err := os.ReadFile("contracts/desktop-v1/source.json")
	if err != nil {
		return err
	}
	var pinned source
	if err = json.Unmarshal(data, &pinned); err != nil {
		return err
	}
	for name, expected := range pinned.SHA256 {
		if name != "protocol.json" && name != "schema.json" && name != "fixtures.json" {
			return errors.New("unexpected contract file")
		}
		data, err = os.ReadFile(filepath.Join("contracts/desktop-v1", name))
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != expected {
			return errors.New("pinned contract bytes mismatch")
		}
	}
	e, err := validate(raw, os.Getenv("RELEASE_VERSION"), os.Getenv("SOURCE_COMMIT"), os.Getenv("LLVY_SERVER_ORIGIN"), pinned)
	if err != nil {
		return err
	}
	if output := os.Getenv("RELEASE_EVIDENCE_OUTPUT"); output != "" {
		data, err = json.MarshalIndent(e, "", "  ")
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
			return err
		}
		return os.WriteFile(output, append(data, '\n'), 0644)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
