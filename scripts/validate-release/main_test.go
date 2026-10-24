package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func validEvidence() (evidence, source) {
	p := source{Commit: strings.Repeat("a", 40), SHA256: map[string]string{}}
	for _, name := range []string{"protocol.json", "schema.json", "fixtures.json"} {
		p.SHA256[name] = strings.Repeat("b", 64)
	}
	e := evidence{SchemaVersion: 1, ClientCommit: strings.Repeat("c", 40), Version: "1.2.3", Origin: "https://app.llvy.my", ContractCommit: p.Commit, ContractSHA256: map[string]string{}, ServerCommit: strings.Repeat("d", 40), DeploymentURL: "https://deployment.llvy.my", Checks: map[string]check{}}
	for k, v := range p.SHA256 {
		e.ContractSHA256[k] = v
	}
	for _, k := range requiredChecks {
		e.Checks[k] = check{"passed", "https://github.com/org/repo/actions/runs/123"}
	}
	return e, p
}
func TestReleaseEvidenceBinding(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*evidence)
	}{
		{"client", func(e *evidence) { e.ClientCommit = strings.Repeat("e", 40) }},
		{"version", func(e *evidence) { e.Version = "1.2.4" }},
		{"origin", func(e *evidence) { e.Origin = "https://another.llvy.my" }},
		{"contract", func(e *evidence) { e.ContractCommit = strings.Repeat("e", 40) }},
		{"checksum", func(e *evidence) { e.ContractSHA256["schema.json"] = strings.Repeat("e", 64) }},
		{"pending", func(e *evidence) { e.Checks["windows10"] = check{"pending", "https://github.com/org/repo/issues/1"} }},
		{"missing", func(e *evidence) { delete(e.Checks, "revocation") }},
		{"placeholder", func(e *evidence) { e.DeploymentURL = "https://server.example.invalid" }},
		{"credential", func(e *evidence) { e.DeploymentURL = "https://secret@llvy.my" }},
		{"deployment query", func(e *evidence) { e.DeploymentURL = "https://llvy.my/?x-vercel-protection-bypass=secret" }},
		{"evidence query", func(e *evidence) {
			e.Checks["windows10"] = check{"passed", "https://github.com/org/repo/issues/1?token=secret"}
		}},
		{"evidence fragment", func(e *evidence) {
			e.Checks["windows10"] = check{"passed", "https://github.com/org/repo/issues/1#secret"}
		}},
		{"server", func(e *evidence) { e.ServerCommit = strings.Repeat("0", 40) }},
	}
	e, p := validEvidence()
	raw, _ := json.Marshal(e)
	if _, err := validate(raw, e.Version, e.ClientCommit, e.Origin, p); err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, p := validEvidence()
			tt.mutate(&e)
			raw, _ := json.Marshal(e)
			if _, err := validate(raw, "1.2.3", strings.Repeat("c", 40), "https://app.llvy.my", p); err == nil {
				t.Fatal("accepted invalid evidence")
			}
		})
	}
	for _, raw := range []string{"", `{}`, string(raw) + ` {}`, strings.Replace(string(raw), `"schemaVersion":1`, `"unexpected":true,"schemaVersion":1`, 1)} {
		if _, err := validate([]byte(raw), e.Version, e.ClientCommit, e.Origin, p); err == nil {
			t.Fatal("accepted incomplete or ambiguous JSON")
		}
	}
}
