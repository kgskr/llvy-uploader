package desktop

import (
	"errors"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ServerOrigin is injected with -ldflags -X. There is no runtime fallback.
var ServerOrigin string

var ErrOrigin = errors.New("서버 주소가 없거나 올바르지 않습니다. LLVY_SERVER_ORIGIN을 지정한 빌드가 필요합니다")
var hostLabel = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

// ValidateOrigin also restricts the value to characters safe in linker flags.
func ValidateOrigin(origin string, development bool) error {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.Opaque != "" || strings.ContainsAny(origin, "?#\\\"' \t\r\n") {
		return ErrOrigin
	}
	if u.Scheme != "https" && !(development && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")) {
		return ErrOrigin
	}
	host := u.Hostname()
	if len(host) > 253 || host == "" {
		return ErrOrigin
	}
	if net.ParseIP(host) == nil {
		for _, label := range strings.Split(host, ".") {
			if !hostLabel.MatchString(label) {
				return ErrOrigin
			}
		}
	}
	if strings.HasSuffix(u.Host, ":") {
		return ErrOrigin
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return ErrOrigin
		}
	}
	return nil
}
