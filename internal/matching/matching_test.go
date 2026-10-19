package matching

import (
	"fmt"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	ids, errs := Parse("kr_000123, KR-123; 000123\n99999999999999999999")
	if len(errs) != 0 || len(ids) != 3 || ids[0].String() != "KR-123" || ids[1].String() != "123" || ids[2].Number != "99999999999999999999" {
		t.Fatalf("%v %v", ids, errs)
	}
	_, errs = Parse("이름#태그 K-123 123456789012345678901 KR-1x")
	if len(errs) != 4 {
		t.Fatal(errs)
	}
	var input strings.Builder
	for i := 0; i < 101; i++ {
		fmt.Fprintf(&input, "%d ", i)
	}
	ids, errs = Parse(input.String())
	if len(ids) != 0 || len(errs) != 1 {
		t.Fatal("over-limit batch must be rejected")
	}
}
func TestFilenameTokens(t *testing.T) {
	for _, tt := range []struct {
		input, name string
		want        bool
	}{
		{"123", "KR-1234.rofl", false}, {"123", "KR-000123.ROFL", true}, {"KR-123", "NA1-123.rofl", false},
		{"123", "copy 123 (2).rofl", true}, {"KR-123", "copy KR-123.rofl", false}, {"123", "abc123.rofl", false},
		{"123", "123abc.rofl", false}, {"123", "123.rofl.bak", false}, {"123", "x-123_4.rofl", true},
		{"123", "12345678901234567890123.rofl", false}, {"0", "000.rofl", true},
	} {
		t.Run(tt.input+"/"+tt.name, func(t *testing.T) {
			ids, _ := Parse(tt.input)
			if got := Matches(ids[0], tt.name); got != tt.want {
				t.Fatalf("got %v", got)
			}
		})
	}
}
