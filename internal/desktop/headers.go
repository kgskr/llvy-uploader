package desktop

import (
	"bytes"
	"encoding/json"
	"strings"
)

// HeaderMap rejects duplicates while decoding, before a Go map could collapse
// identical JSON keys. Validation also rejects case variants and unknown names.
type HeaderMap map[string]string

func (h *HeaderMap) UnmarshalJSON(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return ErrContract
	}
	result := HeaderMap{}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return ErrContract
		}
		name, ok := key.(string)
		if !ok || seen[strings.ToLower(name)] {
			return ErrContract
		}
		seen[strings.ToLower(name)] = true
		var value string
		if err := d.Decode(&value); err != nil {
			return ErrContract
		}
		result[name] = value
	}
	if end, err := d.Token(); err != nil || end != json.Delim('}') {
		return ErrContract
	}
	*h = result
	return nil
}
