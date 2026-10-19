// Package desktopcontract embeds the pinned server policy for offline validation.
package desktopcontract

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"slices"
)

//go:embed protocol.json
var encoded []byte
var policy struct {
	Transfer struct {
		Origin  string                     `json:"origin"`
		Path    string                     `json:"path"`
		Method  string                     `json:"method"`
		Headers map[string]json.RawMessage `json:"headers"`
	} `json:"transfer"`
}

func init() {
	if err := json.Unmarshal(encoded, &policy); err != nil {
		panic("invalid embedded desktop policy")
	}
}
func Endpoint() (string, string, string) {
	return policy.Transfer.Origin, policy.Transfer.Path, policy.Transfer.Method
}
func HeaderCount() int { return len(policy.Transfer.Headers) }

var label = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)

func HeaderAllowed(name, value string) bool {
	raw, ok := policy.Transfer.Headers[name]
	if !ok {
		return false
	}
	var values []string
	if json.Unmarshal(raw, &values) == nil {
		return slices.Contains(values, value)
	}
	var rule string
	return json.Unmarshal(raw, &rule) == nil && rule == "dns-label" && label.MatchString(value)
}
