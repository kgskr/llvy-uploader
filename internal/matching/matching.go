// Package matching treats match numbers as strings, never floating point values.
package matching

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const MaxIDs = 100

type ID struct{ Platform, Number string }

func (id ID) String() string {
	if id.Platform != "" {
		return id.Platform + "-" + id.Number
	}
	return id.Number
}

var inputPattern = regexp.MustCompile(`^(?:([A-Za-z0-9]{2,8})[-_])?([0-9]{1,20})$`)
var standardName = regexp.MustCompile(`^([A-Za-z0-9]{2,8})[-_]([0-9]{1,20})$`)
var numericToken = regexp.MustCompile(`[0-9]+`)

func number(s string) string {
	s = strings.TrimLeft(s, "0")
	if s == "" {
		return "0"
	}
	return s
}

// Parse returns valid unique IDs and separate errors. Exceeding the batch bound
// rejects the whole batch, so a pasted list cannot silently become partial work.
func Parse(input string) ([]ID, []error) {
	var ids []ID
	var errs []error
	seen := make(map[ID]bool)
	for _, token := range strings.FieldsFunc(input, func(r rune) bool { return unicode.IsSpace(r) || r == ',' || r == ';' }) {
		if strings.Contains(token, "#") {
			errs = append(errs, fmt.Errorf("%s: Riot ID가 아닌 경기 번호를 입력하세요", token))
			continue
		}
		m := inputPattern.FindStringSubmatch(token)
		if m == nil {
			errs = append(errs, fmt.Errorf("%s: 경기 번호는 1~20자리 숫자와 선택적 2~8자리 플랫폼 접두어여야 합니다", token))
			continue
		}
		id := ID{strings.ToUpper(m[1]), number(m[2])}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
		if len(ids) > MaxIDs {
			return nil, append(errs, fmt.Errorf("한 번에 최대 %d개의 고유 게임 ID를 입력하세요", MaxIDs))
		}
	}
	return ids, errs
}

// Matches recognizes platform identity only in the standard filename form.
// Nonstandard names can be candidates for unqualified IDs, never proof of a platform.
func Matches(id ID, filename string) bool {
	if !strings.EqualFold(filepath.Ext(filename), ".rofl") {
		return false
	}
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	if m := standardName.FindStringSubmatch(base); m != nil {
		return number(m[2]) == id.Number && (id.Platform == "" || strings.EqualFold(id.Platform, m[1]))
	}
	if id.Platform != "" {
		return false
	}
	for _, loc := range numericToken.FindAllStringIndex(base, -1) {
		// An alphanumeric word is not a complete numeric token.
		before, after := base[:loc[0]], base[loc[1]:]
		if len(before) > 0 {
			r := []rune(before)
			if unicode.IsLetter(r[len(r)-1]) || unicode.IsNumber(r[len(r)-1]) {
				continue
			}
		}
		if len(after) > 0 {
			r := []rune(after)
			if unicode.IsLetter(r[0]) || unicode.IsNumber(r[0]) {
				continue
			}
		}
		token := base[loc[0]:loc[1]]
		if len(token) <= 20 && number(token) == id.Number {
			return true
		}
	}
	return false
}
